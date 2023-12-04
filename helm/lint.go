package helm

import (
	"context"
	"os"

	"dagger.io/dagger"
	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/helper"
	"github.com/urfave/cli/v2"

	"github.com/creasty/defaults"
	"github.com/gookit/validate"
)

type HelmBuildOption struct {
	WithLint             bool   `default:"true"`
	WithProxy            bool   `default:"true"`
	WithPush             bool   `default:"false"`
	WithRegistryUsername string `validate:"validateRegistryAuth"`
	WithRegistryPassword string `validate:"validateRegistryAuth"`
	RegistryName         string `validate:"required"`
	PathContext          string `default:"."`
}

func (h HelmBuildOption) ValidateRegistryAuth(val string) bool {
	if h.WithPush && val == "" {
		return false
	}

	return true
}

// GetBuildCommand permit to get the command spec to add on cli
func GetBuildCommand(registryName string) *cli.Command {
	return &cli.Command{
		Name:  "buildHelmCHart",
		Usage: "Build the chart helm",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "push",
				Usage: "Push chart on registry",
			},
			&cli.StringFlag{
				Name:     "registry-username",
				Usage:    "The username to connect on registry",
				Required: false,
				EnvVars:  []string{"REGISTRY_USERNAME"},
			},
			&cli.StringFlag{
				Name:     "registry-password",
				Usage:    "The password to connect on registry",
				Required: false,
				EnvVars:  []string{"REGISTRY_PASSWORD"},
			},
			&cli.StringFlag{
				Name:    "registry-cert-path",
				Usage:   "The cert full path to connect on internal registry",
				Value:   "",
				EnvVars: []string{"REGISTRY_CERT_PATH"},
			},
			&cli.StringFlag{
				Name:    "path",
				Usage:   "The path of helm chart",
				Value:   ".",
				EnvVars: []string{"REGISTRY_CERT_PATH"},
			},
		},
		Action: func(c *cli.Context) (err error) {
			// initialize Dagger client
			client, err := helper.WithCustomCa(c.Context, c.String("registry-cert-path"), dagger.WithLogOutput(os.Stdout))
			if err != nil {
				panic(err)
			}
			defer client.Close()

			buildOption := &HelmBuildOption{
				RegistryName:         registryName,
				WithPush:             c.Bool("push"),
				WithRegistryUsername: c.String("registry-username"),
				WithRegistryPassword: c.String("registry-password"),
				PathContext:          c.String("path"),
			}

			return BuildHelm(c.Context, client, buildOption)
		},
	}
}

// BuildHelm permit to build helm chart
func BuildHelm(ctx context.Context, client *dagger.Client, option *HelmBuildOption) (err error) {

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	// Lint image if needed
	if option.WithLint {
		_, err := client.
			Container().
			From("alpine/helm:latest").
			WithDirectory("/project", client.Host().Directory(option.PathContext)).
			WithWorkdir("/project").
			WithExec(helper.ForgeCommand("lint .")).
			Stdout(ctx)

		if err != nil {
			return errors.Wrap(err, "Error when lint helm chart")
		}
	}

	return nil
}
