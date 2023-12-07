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

type HelmCmdOption struct {
	Cmd            string `validate:"required"`
	KubeconfigPath string `validate:"required"`
	WithProxy      bool   `default:"true"`
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
