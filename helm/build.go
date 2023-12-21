package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"dagger.io/dagger"
	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/helper"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"

	"github.com/creasty/defaults"
	"github.com/gookit/validate"
)

type BuildOption struct {
	WithProxy            bool   `default:"true"`
	WithPush             bool   `default:"false"`
	WithRegistryUsername string `validate:"validateRegistryAuth"`
	WithRegistryPassword string `validate:"validateRegistryAuth"`
	RegistryUrl          string `validate:"validateRegistryAuth"`
	RepositoryName       string `validate:"validateRegistryAuth"`
	PathContext          string `default:"."`
	CaPath               string
	Version              string
}

func (h BuildOption) ValidateRegistryAuth(val string) bool {
	if h.WithPush && val == "" {
		return false
	}

	return true
}

func InitBuildFlag(app *cli.App) {
	flags := []cli.Flag{
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
			Name:    "custom-ca-path",
			Usage:   "The custom ca full path file",
			EnvVars: []string{"CUSTOM_CA_PATH"},
		},
	}

	app.Flags = append(app.Flags, flags...)
}

// GetBuildCommand permit to get the command spec to add on cli
func GetBuildCommand(registryUrl string, repositoryName string) *cli.Command {
	return &cli.Command{
		Name:  "buildHelmChart",
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
				Name:    "custom-ca-path",
				Usage:   "The custom ca full path file",
				EnvVars: []string{"CUSTOM_CA_PATH"},
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: "The path of helm chart",
				Value: ".",
			},
			&cli.StringFlag{
				Name:  "version",
				Usage: "The chart helm version to build",
			},
		},
		Action: func(c *cli.Context) (err error) {
			// initialize Dagger client
			client, err := dagger.Connect(c.Context, dagger.WithLogOutput(os.Stdout))
			if err != nil {
				panic(err)
			}
			defer client.Close()

			buildOption := &BuildOption{
				RegistryUrl:          registryUrl,
				RepositoryName:       repositoryName,
				WithPush:             c.Bool("push"),
				WithRegistryUsername: c.String("registry-username"),
				WithRegistryPassword: c.String("registry-password"),
				PathContext:          c.String("path"),
				CaPath:               c.String("custom-ca-path"),
				Version:              c.String("version"),
			}

			return BuildHelm(c.Context, client, buildOption)
		},
	}
}

// BuildHelm permit to build helm chart
func BuildHelm(ctx context.Context, client *dagger.Client, option *BuildOption) (err error) {

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	// Read chart file if need to push or need to create new version
	dataChart := make(map[string]any)
	if option.Version != "" || option.WithPush {
		// Read chart file
		yfile, err := os.ReadFile("Chart.yaml")
		if err != nil {
			return errors.Wrap(err, "Error when read file Chart.yaml")
		}

		if err = yaml.Unmarshal(yfile, &dataChart); err != nil {
			return errors.Wrap(err, "Error when decode YAML file")
		}

		if option.Version != "" {
			dataChart["version"] = option.Version
			yfile, err = yaml.Marshal(dataChart)
			if err != nil {
				return errors.Wrap(err, "Error when encode YAML file")
			}
			if err = os.WriteFile("Chart.yaml", yfile, 0644); err != nil {
				return errors.Wrap(err, "Error when write Chart.yaml")
			}
		}
	}

	container := getHelmContainer(client, option.PathContext)

	if option.CaPath != "" {
		// Copy the certificate in temporary folder because of the are issue with buildkit when file is symlink
		caTmpFile, err := os.CreateTemp("", "ca")
		if err != nil {
			return errors.Wrap(err, "Error when create temporary file to store CA content")
		}
		defer os.Remove(caTmpFile.Name())

		caContent, err := os.ReadFile(option.CaPath)
		if err != nil {
			return errors.Wrap(err, "Error when read CA file")
		}
		if _, err = caTmpFile.Write(caContent); err != nil {
			return errors.Wrap(err, "Error when write CA contend")
		}

		container = container.WithMountedFile(fmt.Sprintf("/etc/ssl/certs/%s", filepath.Base(option.CaPath)), client.Host().File(caTmpFile.Name()))
	}

	// package helm
	container = container.WithExec(helper.ForgeCommand("package -u ."))

	// push helm chart
	if option.WithPush {

		// Login to registry
		registryUsername := client.SetSecret("registry-username", option.WithRegistryUsername)
		registryPassword := client.SetSecret("registry-password", option.WithRegistryPassword)

		container = container.
			WithSecretVariable("REGISTRY_USERNAME", registryUsername).
			WithSecretVariable("REGISTRY_PASSWORD", registryPassword).
			WithEntrypoint([]string{"/bin/sh", "-c"}).
			WithExec([]string{fmt.Sprintf("helm registry login -u $REGISTRY_USERNAME -p $REGISTRY_PASSWORD %s", option.RegistryUrl)})

		// Push to registry
		container = container.WithExec([]string{fmt.Sprintf("helm push %s-%s.tgz oci://%s/%s", dataChart["name"], dataChart["version"], option.RegistryUrl, option.RepositoryName)})
	}

	_, err = container.Stdout(ctx)
	if err != nil {
		return errors.Wrap(err, "Error when package and push helm chart")
	}

	return nil
}
