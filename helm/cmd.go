package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"dagger.io/dagger"
	"emperror.dev/errors"
	"github.com/creasty/defaults"
	"github.com/disaster37/dagger-library-go/helper"
	"github.com/gookit/validate"
	"github.com/urfave/cli/v2"
)

type HelmCmdOption struct {
	Cmd            string `validate:"required"`
	KubeconfigPath string `validate:"required"`
	WithProxy      bool   `default:"true"`
	CaPath         string
}

// GetBuildCommand permit to get the command spec to add on cli
func GetCmdCommand() *cli.Command {
	return &cli.Command{
		Name:  "helm",
		Usage: "Run helm command",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "kubeconfig",
				Usage:   "The kube config path",
				EnvVars: []string{"KUBECONFIG"},
			},
			&cli.StringFlag{
				Name:     "cmd",
				Usage:    "the helm command",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "custom-ca-path",
				Usage:   "The custom ca full path file",
				EnvVars: []string{"CUSTOM_CA_PATH"},
			},
		},
		Action: func(c *cli.Context) (err error) {
			// initialize Dagger client
			client, err := dagger.Connect(c.Context, dagger.WithLogOutput(os.Stdout))
			if err != nil {
				panic(err)
			}
			defer client.Close()

			cmdOption := &HelmCmdOption{
				Cmd:            c.String("cmd"),
				KubeconfigPath: c.String("kubeconfig"),
				CaPath:         c.String("custom-ca-path"),
			}

			return HelmCommand(c.Context, client, cmdOption)
		},
	}
}

// HelmCommand permit to run any helm command
func HelmCommand(ctx context.Context, client *dagger.Client, option *HelmCmdOption) (err error) {

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	container := client.
		Container().
		From("alpine/helm:latest")

	if option.WithProxy {
		container = helper.WithProxy(container)
	}

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

	container.
		WithDirectory("/project", client.Host().Directory(".")).
		WithMountedFile("/tmp/kubeconfig", client.Host().File(option.KubeconfigPath)).
		WithEnvVariable("KUBECONFIG", "/tmp/kubeconfig").
		WithWorkdir("/project").
		WithExec(helper.ForgeCommand(option.Cmd)).
		Stdout(ctx)

	if err != nil {
		return errors.Wrap(err, "Error when execute helm command")
	}

	return nil
}
