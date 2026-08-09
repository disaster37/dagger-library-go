// A generated module for Codecov functions
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
	"dagger/codecov/internal/dagger"
	"dagger/codecov/internal/uploadcmd"
	"fmt"
)

type Codecov struct {
	// The container
	Container *dagger.Container
}

// New initializes the codecov dagger module
func New(
	ctx context.Context,
	// A custom base image containing a codecov uploader
	// +optional
	base *dagger.Container,
	// The codecov uploader version to download (default: latest)
	// +optional
	version string,
) (*Codecov, error) {
	urlCodecov := "https://uploader.codecov.io/latest/linux/codecov"
	if version != "" {
		urlCodecov = fmt.Sprintf("https://uploader.codecov.io/v%s/linux/codecov", version)
	}

	codeCov := base
	if base == nil {
		codeCov = dag.Container().
			From("cgr.dev/chainguard/wolfi-base").
			WithExec([]string{"apk", "add", "--update", "curl", "git"}).
			WithExec([]string{"curl", "-fL", "-o", "/bin/codecov", "-s", urlCodecov}).
			WithExec([]string{"chmod", "+x", "/bin/codecov"}).
			WithExec([]string{"/bin/codecov", "--version"})
	}

	return &Codecov{
		Container: codeCov.WithWorkdir("/project"),
	}, nil
}

// WithContainer permit to set container
func (h *Codecov) WithContainer(ctn *dagger.Container) *Codecov {
	h.Container = ctn
	return h
}

// Upload uploads coverage reports to Codecov. The token is passed only via
// the CODECOV_TOKEN secret environment variable; the uploader binary is
// executed directly so argument values reach it unmodified.
func (h *Codecov) Upload(
	ctx context.Context,

	// The source directory
	src *dagger.Directory,

	// The codecov token
	token *dagger.Secret,

	// +optional
	name string, // optional name

	// +optional
	files []string, // optional list of coverage files

	// +optional
	flags []string, // optional additional flags for uploader
) (string, error) {
	cmd := uploadcmd.Build(name, files, flags)

	return h.Container.
		WithDirectory("/project", src).
		WithSecretVariable("CODECOV_TOKEN", token).
		WithExec(cmd).Stdout(ctx)
}
