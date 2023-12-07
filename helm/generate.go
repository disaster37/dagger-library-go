package helm

import (
	"context"
	"fmt"
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
	FileName    string `validate:"required"`
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
			&cli.StringFlag{
				Name:  "schema-file",
				Usage: "The schema file name",
				Value: "values.schema.json",
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
				FileName:    c.String("schema-file"),
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

	getGeneratorContainer(client, option).
		WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -s %s --values values.yaml", option.FileName))).
		Directory(".").
		Export(ctx, option.PathContext)

	if err != nil {
		return errors.Wrap(err, "Error when generate helm schema")
	}

	return nil
}

// GetBuildCommand permit to get the command spec to add on cli
func GetGenerateReadmeCommand() *cli.Command {
	return &cli.Command{
		Name:  "generateHelmReadme",
		Usage: "Generate the helm readme",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Usage:   "The path of helm chart",
				Value:   ".",
				EnvVars: []string{"REGISTRY_CERT_PATH"},
			},
			&cli.StringFlag{
				Name:  "readme-file",
				Usage: "The readme file name to generate",
				Value: "README.md",
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
				FileName:    c.String("readme-file"),
			}

			return GenerateHelmReadme(c.Context, client, generateOption)
		},
	}
}

// BuildHelm permit to build helm chart
func GenerateHelmReadme(ctx context.Context, client *dagger.Client, option *HelmGenerateOption) (err error) {

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	getGeneratorContainer(client, option).
		WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -r %s --values values.yaml", option.FileName))).
		Directory(".").
		Export(ctx, option.PathContext)

	if err != nil {
		return errors.Wrap(err, "Error when generate helm readme")
	}

	return nil
}

func getGeneratorContainer(client *dagger.Client, option *HelmGenerateOption) *dagger.Container {
	container := client.
		Container().
		From("node:21-alpine")

	if option.WithProxy {
		container = helper.WithProxy(container)
	}

	return container.
		WithDirectory("/project", client.Host().Directory(option.PathContext)).
		WithWorkdir("/project").
		WithExec(helper.ForgeCommand("npm install -g @bitnami/readme-generator-for-helm"))
}
