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
	"os"
	"strings"
)

type Codecov struct {
	// The container
	Container *dagger.Container

	// The source directory
	// +private
	Src *dagger.Directory
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
	// a path to a directory containing the source code
	// +required
	src *dagger.Directory,
) (*Codecov, error) {

	var (
		codeCov    *dagger.Container
		urlCodecov string
	)

	if version != "" {
		urlCodecov = fmt.Sprintf("https://uploader.codecov.io/v%s/linux/codecov", version)
	} else {
		urlCodecov = "https://cli.codecov.io/latest/linux/codecov"
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
		WithDirectory("/project", src).
		WithWorkdir("/project")

	return &Codecov{
		Container: codeCov,
		Src:       src,
	}, nil
}

// WithContainer permit to set container
func (h *Codecov) WithContainer(ctn *dagger.Container) *Codecov {
	h.Container = ctn
	return h
}

func (h *Codecov) Upload(
	ctx context.Context,

	// The codecov token
	token *dagger.Secret,

	// Inject all variable environment on Codecov container to auto discover them by Codecov upload
	// +optional
	injectCiEnvironment bool,

	// +optional
	name string, // optional name
	// +optional
	verbose bool, // optional verbose output
	// +optional
	files []string, // optional list of coverage files
	// +optional
	flags []string, // optional additional flags for uploader
) (string, error) {
	cmd := []string{"/bin/codecov", "-t", "$CODECOV_TOKEN"}

	if name != "" {
		cmd = append(cmd, "-n", name)
	}

	if verbose {
		cmd = append(cmd, "-v")
	}

	if len(files) > 0 {
		cmd = append(cmd, "-f")
		cmd = append(cmd, files...)
	}

	if len(flags) > 0 {
		cmd = append(cmd, flags...)
	}

	// Inject all current env on codecov container
	if injectCiEnvironment {
		for _, env := range os.Environ() {
			if i := strings.Index(env, "="); i >= 0 {
				h.Container = h.Container.WithEnvVariable(env[:i], env[i+1:])
			}
		}
	}

	return h.Container.WithExec(cmd).Stdout(ctx)
}
