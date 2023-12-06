package helm

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/helper"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"

	"github.com/creasty/defaults"
	"github.com/gookit/validate"
)

type HelmBuildOption struct {
	WithLint             bool   `default:"true"`
	WithProxy            bool   `default:"true"`
	WithPush             bool   `default:"false"`
	WithRegistryUsername string `validate:"validateRegistryAuth"`
	WithRegistryPassword string `validate:"validateRegistryAuth"`
	RegistryUrl          string `validate:"required"`
	PathContext          string `default:"."`
}

func (h HelmBuildOption) ValidateRegistryAuth(val string) bool {
	if h.WithPush && val == "" {
		return false
	}

	return true
}

// GetBuildCommand permit to get the command spec to add on cli
func GetBuildCommand(registryUrl string) *cli.Command {
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
				RegistryUrl:          registryUrl,
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

	container := client.
		Container().
		From("alpine/helm:latest").
		WithMountedDirectory("/etc/ssl/certs", client.Host().Directory("/etc/ssl/certs")).
		WithDirectory("/project", client.Host().Directory(option.PathContext)).
		WithWorkdir("/project")

	// Lint image if needed
	if option.WithLint {
		_, err = container.
			WithExec(helper.ForgeCommand("lint .")).
			Stdout(ctx)
		if err != nil {
			return errors.Wrap(err, "Error when lint helm chart")
		}
	}

	// package helm
	container = container.WithExec(helper.ForgeCommand("package -u ."))

	// push helm chart
	if option.WithPush {
		if option.RegistryUrl == "" {
			return errors.New("You need to set the registry URL")
		}

		// Login to registry
		if option.WithRegistryUsername != "" && option.WithRegistryPassword != "" {
			container = container.WithExec(helper.ForgeCommand(fmt.Sprintf("registry login -u %s -p %s", option.WithRegistryUsername, option.WithRegistryPassword)))
		}

		// Get the current version
		yfile, err := os.ReadFile("Chart.yaml")
		if err != nil {
			return errors.Wrap(err, "Error when read file Chart.yaml")
		}
		data := make(map[string]any)
		if err = yaml.Unmarshal(yfile, &data); err != nil {
			return errors.Wrap(err, "Error when decode YAML file")
		}

		// Push to registry
		container = container.WithExec(helper.ForgeCommand(fmt.Sprintf("push hms-%s.tgz oci://%s", data["version"], option.RegistryUrl)))
	}

	_, err = container.Stdout(ctx)
	if err != nil {
		return errors.Wrap(err, "Error when package and push helm chart")
	}

	return nil
}
