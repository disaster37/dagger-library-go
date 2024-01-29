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
)

type HelmCmdOption struct {
	Cmd            string `validate:"required"`
	KubeconfigPath string `validate:"required"`
	WithProxy      bool   `default:"true"`
	CaPath         string
}

// HelmCommand permit to run any helm command
func HelmCommand(ctx context.Context, client *dagger.Client, option *HelmCmdOption) (err error) {

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	image := fmt.Sprintf("alpine/helm:%s", helm_version)

	container := client.
		Container().
		From(image)

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
