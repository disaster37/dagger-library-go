// A generated module for DevelopGo functions
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

import "dagger/develop-go/internal/dagger"

type DevelopGo struct {
	// +private
	Golang *dagger.GolangCodeWorkspace
	// +private
	Src *dagger.Directory
	// +private
	Repo string
	// +private
	DocPath string
	// +private
	ContributingFilePath string
	// +private
	ReviewConstraints string
}

func New(
	// +optional
	// +defaultPath="/"
	// +ignore=[".git", "**/node_modules"]
	source *dagger.Directory,
	// +required
	repo string,
	// +optional
	// +default="docs"
	docPath string,
	// +optional
	// +default="CONTRIBUTING.md"
	contributeFilePath string,
	// +optional
	reviewConstraints string,
	// +optional
	// extra golang container
	container *dagger.Container,
) *DevelopGo {
	return &DevelopGo{
		Src:                  source,
		Repo:                 repo,
		DocPath:              docPath,
		ContributingFilePath: contributeFilePath,
		ReviewConstraints:    reviewConstraints,
		Golang: dag.GolangCodeWorkspace(
			dagger.GolangCodeWorkspaceOpts{
				Source:    source,
				Container: container,
			},
		),
	}
}
