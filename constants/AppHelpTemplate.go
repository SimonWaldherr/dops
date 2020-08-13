package constants

import (
	"fmt"
	"github.com/dops-cli/dops/say/color"
)

// AppHelpTemplate contains the template of dops help text.
var AppHelpTemplate = fmt.Sprintf(color.HiCyanString("\nDops - CLI DevOps Toolkit") + `

{{if .VisibleFlags}}` + color.New(color.FgHiYellow, color.Underline).Sprint(`Global options`) + `
  ` + color.YellowString(`{{range $index, $option := .VisibleFlags}}{{if $index}}`) + `
  ` + color.YellowString(`{{end}}{{$option}}{{end}}{{end}}`) + `

{{if .VisibleCommands}}` + color.New(color.FgHiYellow, color.Underline).Sprint(`Modules`) + `{{range .VisibleCategories}}{{if .Name}}
  [` + color.HiCyanString(`{{.Name}}`) + `]{{range .VisibleCommands}}
    · ` + color.HiMagentaString(`{{join .Names ", "}}`) + color.HiRedString(`{{"\t|\t"}}`) + `{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
    · ` + color.HiMagentaString(`{{join .Names ", "}}`) + color.HiRedString(`{{"\t|\t"}}`) + `{{.Usage}}{{end}}{{end}}{{end}}{{end}}

` + color.HiRedString("Contribute to this tool here: https://github.com/dops-cli ") + color.RedString("<3\n"))