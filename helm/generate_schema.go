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

type GenerateSchemaOption struct {
	WithProxy   bool   `default:"true"`
	PathContext string `default:"."`
	FileName    string `default:"values.schema.json"`
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
			client, err := dagger.Connect(c.Context, dagger.WithLogOutput(os.Stdout))
			if err != nil {
				panic(err)
			}
			defer client.Close()

			generateOption := &GenerateSchemaOption{
				PathContext: c.String("path"),
				FileName:    c.String("schema-file"),
			}

			return GenerateSchema(c.Context, client, generateOption)
		},
	}
}

// BuildHelm permit to build helm chart
func GenerateSchema(ctx context.Context, client *dagger.Client, option *GenerateSchemaOption) (err error) {

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	_, err = getGeneratorContainer(client, option.PathContext, option.WithProxy).
		WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -s %s --values values.yaml", option.FileName))).
		File(option.FileName).
		Export(ctx, fmt.Sprintf("%s/%s", option.PathContext, option.FileName))

	if err != nil {
		return errors.Wrap(err, "Error when generate helm schema")
	}

	return nil
}
