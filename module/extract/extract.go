package extract

import (
	"io/ioutil"
	"os"
	"regexp"

	"github.com/urfave/cli/v2"

	"github.com/dops-cli/dops/categories"
	"github.com/dops-cli/dops/say"
	"github.com/dops-cli/dops/utils"
)

type Module struct{}

func (Module) GetCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:        "extract-text",
			Usage:       "Extracts text using regex from a file",
			Description: `Extract-text can be used to extract text from a file using regex patterns.`,
			Category:    categories.TextProcessing,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "regex",
					Aliases: []string{"r"},
					Usage:   "extracts matching strings with `PATTERN`",
				},
				&cli.PathFlag{
					Name:      "input",
					Aliases:   []string{"i"},
					Usage:     "use `FILE` as input",
					TakesFile: true,
				},
				&cli.StringFlag{
					Name:    "output",
					Aliases: []string{"o"},
					Usage:   "outputs to directory `DIR`",
				},
			},
			Action: func(c *cli.Context) error {
				regex := c.String("regex")
				input := c.Path("input")
				output := c.String("output")

				var foundStrings []string

				r, err := regexp.Compile(regex)
				if err != nil {
					return err
				}

				foundStrings = r.FindAllString(utils.FileOrStdin(input), -1)

				if output == "" {
					for _, s := range foundStrings {
						say.Text(s)
					}
				} else {
					var out string
					for _, s := range foundStrings {
						out += s + "\n"
					}
					err := ioutil.WriteFile(output, []byte(out), os.ModeAppend)
					if err != nil {
						return err
					}
				}
				return nil
			},
		},
	}
}
