package debug

import (
	"github.com/urfave/cli/v2"
)

// IsDebug returns true if dops is run in debugging mode.
var IsDebug bool

type Flag struct{}

func (Flag) GetFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "debug",
			Value:       false,
			Destination: &IsDebug,
		},
	}
}