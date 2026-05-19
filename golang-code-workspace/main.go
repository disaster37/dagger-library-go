// A generated module for GolangCodeWorkspace functions
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
	"dagger/golang-code-workspace/internal/dagger"
	"strings"

	"emperror.dev/errors"
)

type GolangCodeWorkspace struct {
	// +private
	Golang *dagger.Golang
}

// NewGolangCodeWorkspace return the Golang code workspace
func New(
	// +optional
	// +defaultPath="/"
	// +ignore=[".git", "**/node_modules"]
	source *dagger.Directory,
	// +optional
	// extra golang container
	container *dagger.Container,
) *GolangCodeWorkspace {
	return &GolangCodeWorkspace{
		Golang: dag.Golang(
			source,
			dagger.GolangOpts{
				Base: container,
			},
		),
	}
}

func (h *GolangCodeWorkspace) LintProject(ctx context.Context, source *dagger.Directory) (string, error) {
	return h.Golang.
		WithSource(source).
		Lint(ctx, true)
}

func (h *GolangCodeWorkspace) RunTest(ctx context.Context, source *dagger.Directory) (string, error) {
	var sb strings.Builder

	coverage, err := h.Golang.
		WithSource(source).
		Test(
			dagger.GolangTestOpts{
				Short:         false,
				Shuffle:       false,
				WithGotestsum: true,
			},
		).Sync(ctx)

	if err != nil {
		return "", err
	}

	sb.WriteString("Code coverage: \n")
	str, err := coverage.Contents(ctx)
	if err != nil {
		return "", errors.Wrap(err, "error when read code coverage file")
	}
	sb.WriteString(str)

	return sb.String(), nil

}

func (h *GolangCodeWorkspace) RunVulnCheck(ctx context.Context, source *dagger.Directory) (string, error) {
	return h.Golang.
		WithSource(source).
		Vulncheck(ctx)
}

func (h *GolangCodeWorkspace) FormatProject(source *dagger.Directory) *dagger.Directory {
	return h.Golang.
		WithSource(source).
		Format()
}

func (h *GolangCodeWorkspace) BuildProject(ctx context.Context, source *dagger.Directory) (string, error) {
	_, err := h.Golang.
		WithSource(source).
		Build(
			dagger.GolangBuildOpts{
				Main: ".",
			},
		).
		Sync(ctx)

	if err != nil {
		return "", err
	}

	return "Build run successfully", nil
}
