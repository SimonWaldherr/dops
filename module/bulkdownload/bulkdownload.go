package bulkdownload

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/pterm/pterm"

	"github.com/dops-cli/dops/categories"
	"github.com/dops-cli/dops/cli"
)

var wg sync.WaitGroup

// Module returns the created module
type Module struct{}

// GetModuleCommands returns the commands of the module
func (Module) GetModuleCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "bulkdownload",
			Usage: "Download multiple files from a list",
			Description: `Bulkdownload downloads all files from a list. 
You can set how many files should be downloaded concurrently..`,
			Category: categories.Web,
			Aliases:  []string{"bd"},
			Action: func(c *cli.Context) error {
				inputFile := c.String("input")
				outputDir := c.String("output")
				concurrentDownloads := c.Int("concurrent")

				urls, err := readLines(inputFile)
				if err != nil {
					return err
				}
				wg.Add(len(urls))

				pterm.Info.Println("Downloading " + pterm.LightMagenta(len(urls)) + " files")

				pb := pterm.DefaultProgressbar.WithTotal(len(urls)).WithTitle("Downloading").Start()

				downloadMultipleFiles(urls, outputDir, concurrentDownloads, pb)
				wg.Wait()
				return nil
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "input",
					Aliases: []string{"i"},
					Usage:   "load URLs from `FILE`",
					Value:   "urls.txt",
				},
				&cli.StringFlag{
					Name:        "output",
					Aliases:     []string{"o"},
					Usage:       "save the downloaded files to `DIR`",
					DefaultText: "current directory",
				},
				&cli.IntFlag{
					Name:    "concurrent",
					Aliases: []string{"c"},
					Usage:   "downloads `NUMBER` files concurrently",
					Value:   3,
				},
			},
			Examples: []cli.Example{
				{
					ShortDescription: "Download all files from urls.txt, with 5 concurrent connections, to the current directory.",
					Usage:            "dops bulkdownload -i urls.txt -c 5",
				},
			},
		},
	}
}

func downloadMultipleFiles(urls []string, outputDir string, concurrentDownloads int, pb *pterm.Progressbar) {

	guard := make(chan struct{}, concurrentDownloads)

	for index, URL := range urls {
		guard <- struct{}{}
		go func(URL string, outputDir string, index int) {
			pb.Title = filepath.Base(URL)
			err := downloadFile(URL, outputDir)
			if err != nil {
				pterm.Fatal.Println(err)
			}
			pterm.Success.Println("Downloaded " + URL)
			pb.Increment()
			<-guard
		}(URL, outputDir, index)
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func downloadFile(URL string, outputDir string) error {

	response, err := http.Get(URL) //nolint:gosec
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		pterm.Error.Println("Downloading " + pterm.Cyan(URL) + " failed with status code: " + pterm.Red(response.StatusCode))
	}

	file := filepath.Base(URL)

	if outputDir != "" {

		err = os.MkdirAll(outputDir, 0770)
		if err != nil {
			return err
		}

		outputDir += string(os.PathSeparator)
	}

	out, err := os.Create(filepath.FromSlash(outputDir) + file)
	if err != nil {
		return err
	}
	defer out.Close()

	// copy from proxyReader
	_, err = io.Copy(out, response.Body)
	if err != nil {
		return err
	}

	wg.Done()
	return nil
}
