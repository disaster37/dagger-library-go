package helper

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func InitDaggerApp() *cli.App {
	app := cli.NewApp()
	app.Usage = "Dagger CI"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Display debug output",
		},
		&cli.BoolFlag{
			Name:  "no-color",
			Usage: "No print color",
		},
		&cli.BoolFlag{
			Name:  "ci",
			Usage: "Run dagger from CI tools",
		},
		&cli.StringFlag{
			Name:    "tag",
			Usage:   "The current git tag",
			EnvVars: []string{"TAG"},
		},
	}

	app.Before = func(c *cli.Context) error {

		if c.Bool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}

		if !c.Bool("no-color") {
			formatter := new(prefixed.TextFormatter)
			formatter.FullTimestamp = true
			formatter.ForceFormatting = true
			logrus.SetFormatter(formatter)
		}

		return nil
	}

	return app
}
