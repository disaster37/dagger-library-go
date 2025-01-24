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
	"fmt"
	"strings"
)

type Codecov struct {
	// The container
	Container *dagger.Container
}

// New initializes the golang dagger module
func New(
	ctx context.Context,
	// A custom base image containing a codecov uploader
	// +optional
	base *dagger.Container,
	// The golang version to use when no go.mod
	// +optional
	version string,
) (*Codecov, error) {

	var (
		codeCov    *dagger.Container
		urlCodecov string
	)

	if version != "" {
		urlCodecov = fmt.Sprintf("https://uploader.codecov.io/v%s/linux/codecov", version)
	} else {
		urlCodecov = "https://uploader.codecov.io/latest/linux/codecov"
	}

	if base != nil {
		codeCov = base
	} else {
		codeCov = dag.Container().
			From("cgr.dev/chainguard/wolfi-base").
			WithExec([]string{"apk", "add", "curl", "git"}).
			WithExec([]string{"curl", "-o", "/bin/codecov", "-s", urlCodecov}).
			WithExec([]string{"chmod", "+x", "/bin/codecov"}).
			WithExec([]string{"ls", "-lah", "/bin/codecov"})
	}

	codeCov = codeCov.
		WithWorkdir("/project")

	return &Codecov{
		Container: codeCov,
	}, nil
}

// WithContainer permit to set container
func (h *Codecov) WithContainer(ctn *dagger.Container) *Codecov {
	h.Container = ctn
	return h
}

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
	cmd := []string{"/bin/codecov", "-t", "$CODECOV_TOKEN", "-v"}

	if name != "" {
		cmd = append(cmd, "-n", name)
	}

	if len(files) > 0 {
		cmd = append(cmd, "-f")
		cmd = append(cmd, files...)
	}

	if len(flags) > 0 {
		cmd = append(cmd, flags...)
	}

	return h.Container.
		WithDirectory("/project", src).
		WithSecretVariable("CODECOV_TOKEN", token).
		WithExec([]string{"sh", "-c", strings.Join(cmd, " ")}).Stdout(ctx)
}
