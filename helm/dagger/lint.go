package main

import (
	"context"

	"github.com/creasty/defaults"
	"github.com/disaster37/dagger-library-go/helm/dagger/internal/dagger"
	"github.com/disaster37/dagger-library-go/helper"
	"github.com/gookit/validate"
)

type LintOption struct {
	Source    *dagger.Directory
	WithImage string `default:"alpine/helm:3.14.3"`
}

// Lint permit to lint helm chart
func (m *Helm) Lint(
	ctx context.Context,

	// the source directory
	source *dagger.Directory,

	// the alternative image
	// +optional
	withImage *string,

	// Files to inject on containers
	// +optional
	withFiles []*dagger.File,
) (stdout string, err error) {

	option := &LintOption{
		Source: source,
	}
	if withImage != nil {
		option.WithImage = *withImage
	}

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	container := m.GetHelmContainer(ctx, option.Source, option.WithImage)
	for _, file := range withFiles {
		fileName, err := file.Name(ctx)
		if err != nil {
			return "", err
		}
		container = container.WithFile(fileName, file)
	}

	return container.
		WithExec(helper.ForgeCommand("helm dependency update")).
		WithExec(helper.ForgeCommand("helm lint .")).
		Stdout(ctx)
}
