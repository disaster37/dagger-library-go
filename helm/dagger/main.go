// A generated module for Helm functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"

	"github.com/disaster37/dagger-library-go/helm/dagger/internal/dagger"
	"github.com/disaster37/dagger-library-go/helper"
)

type Helm struct{}

func (m *Helm) GetGeneratorContainer(ctx context.Context, source *dagger.Directory, withImage string) *dagger.Container {
	return dag.Container().
		From(withImage).
		WithDirectory("/project", source).
		WithWorkdir("/project").
		WithExec(helper.ForgeCommand("npm install -g @bitnami/readme-generator-for-helm"))
}

func (m *Helm) GetHelmContainer(ctx context.Context, source *dagger.Directory, withImage string) *dagger.Container {
	return dag.Container().
		From(withImage).
		WithDirectory("/project", source).
		WithWorkdir("/project")
}

func (m *Helm) GetYQContainer(ctx context.Context, source *dagger.Directory, withImage string) *dagger.Container {
	return dag.Container().
		From(withImage).
		WithDirectory("/project", source).
		WithWorkdir("/project")
}
