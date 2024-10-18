// A generated module for Image functions
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
	"dagger/image/internal/dagger"
	"fmt"
)

type Image struct {
	// +private
	BaseHadolintContainer *dagger.Container

	// +private
	BuildContainer *dagger.Container
}

func New(
	// base hadolint container
	// It need contain hadolint
	// +optional
	baseHadolintContainer *dagger.Container,

	// The external build of container
	// Usefull when need build args
	// +optional
	buildContainer *dagger.Container,
) *Image {
	image := &Image{
		BuildContainer: buildContainer,
	}

	if baseHadolintContainer != nil {
		image.BaseHadolintContainer = baseHadolintContainer
	} else {
		image.BaseHadolintContainer = image.GetBaseHadolintContainer()
	}

	return image
}

// GetBaseHadolintContainer return the default image for hadolint
func (m *Image) GetBaseHadolintContainer() *dagger.Container {
	return dag.Container().
		From("ghcr.io/hadolint/hadolint:2.12.0")
}

// Build permit to build image from Dockerfile
func (m *Image) Build(

	// the source directory
	source *dagger.Directory,

	// The dockerfile path
	// +optional
	// +default="Dockerfile"
	dockerfile string,

	// Set extra directories
	// +optional
	withDirectories []*dagger.Directory,
) *ImageBuild {

	if m.BuildContainer != nil {
		return &ImageBuild{
			Container: m.BuildContainer,
		}
	}

	for _, directory := range withDirectories {
		source = source.WithDirectory(fmt.Sprintf("%s", directory), directory)
	}

	return &ImageBuild{
		Container: source.DockerBuild(
			dagger.DirectoryDockerBuildOpts{
				Dockerfile: dockerfile,
			},
		),
	}
}
