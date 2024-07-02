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

type LintOption struct {
	WithProxy   bool   `default:"true"`
	PathContext string `default:"."`
	WithFiles   map[string]*dagger.File
	CaPath      string
	WithImage   string `default:"alpine/helm:3.14.3"`
}

// Lint permit to lint helm
func Lint(ctx context.Context, client *dagger.Client, option *LintOption) (err error) {

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	container := getHelmContainer(client, option.WithImage, option.PathContext, option.WithProxy)

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

	container = container.
		WithExec(helper.ForgeCommand("dependency update")).
		WithExec(helper.ForgeCommand("lint ."))

	for fileName, file := range option.WithFiles {
		container = container.WithFile(fileName, file)
	}

	_, err = container.
		Stdout(ctx)
	if err != nil {
		return errors.Wrap(err, "Error when lint helm chart")
	}

	return nil
}
