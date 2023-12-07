package helm

import (
	"context"
	"os"

	"dagger.io/dagger"
	"emperror.dev/errors"
	"github.com/creasty/defaults"
	"github.com/disaster37/dagger-library-go/helper"
	"github.com/gookit/validate"
	"github.com/urfave/cli/v2"
)

type HelmGenerateOption struct {
	WithProxy   bool   `default:"true"`
	PathContext string `default:"."`
}

// GetBuildCommand permit to get the command spec to add on cli
func GetGenerateSchemaCommand() *cli.Command {
	return &cli.Command{
		Name:  "generateHelmSchema",
		Usage: "Generate the helm schema",
		Flags: []cli.Flag{
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

			generateOption := &HelmGenerateOption{
				PathContext: c.String("path"),
			}

			return GenerateHelmSchema(c.Context, client, generateOption)
		},
	}
}

// BuildHelm permit to build helm chart
func GenerateHelmSchema(ctx context.Context, client *dagger.Client, option *HelmGenerateOption) (err error) {

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	container := client.
		Container().
		From("node:21-alpine")

	if option.WithProxy {
		container = helper.WithProxy(container)
	}

	_, err = container.
		WithDirectory("/project", client.Host().Directory(option.PathContext)).
		WithWorkdir("/project").
		WithExec(helper.ForgeCommand("npm install -g @bitnami/readme-generator-for-helm")).
		WithExec(helper.ForgeCommand("readme-generator -s values.schema.json --values values.yaml")).
		Directory(".").
		Export(ctx, option.PathContext)

	if err != nil {
		return errors.Wrap(err, "Error when generate helm schema")
	}

	return nil
}
