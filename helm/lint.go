package main

import (
	"context"

	"dagger/helm/internal/dagger"

	"github.com/creasty/defaults"
	"github.com/disaster37/dagger-library-go/lib/helper"
	"github.com/gookit/validate"
)

type LintOption struct {
	Source *dagger.Directory `validate:"required"`
}

// Lint permit to lint helm chart
func (m *Helm) Lint(
	ctx context.Context,

	// the source directory
	source *dagger.Directory,

	// Files to inject on containers
	// +optional
	withFiles []*dagger.File,
) (stdout string, err error) {

	option := &LintOption{
		Source: source,
	}

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	container := m.baseHelmContainer.
		WithDirectory("/project", source).
		WithWorkdir("/project")

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
