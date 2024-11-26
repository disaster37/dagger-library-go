package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"
	"strings"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/helper"
)

const (
	goMod     = "go.mod"
	goWorkDir = "/src"
)

type Golang struct {

	// +private
	GolangModule *dagger.Golang

	// +private
	Src *dagger.Directory
}

func NewGolang(
	ctx context.Context,

	// The source directory
	// +required
	src *dagger.Directory,

	// Container to use with operator-sdk cli inside and golang
	// +optional
	container *dagger.Container,
) *Golang {

	// Compute the golang base container version
	return &Golang{
		GolangModule: dag.Golang(src, dagger.GolangOpts{Base: container}),
		Src:          src,
	}
}

// Container return the Golang container
func (h *Golang) Container() *dagger.Container {
	return h.GolangModule.Container()
}

func (h *Golang) Oci() *Oci {
	return NewOci(h.Container)
}

// Lint the target project using golangci-lint
func (h *Golang) Lint(
	ctx context.Context,
	// the type of report that should be generated
	// +optional
	// +default="colored-line-number"
	format string,
) (string, error) {
	ctr := h.Container
	if _, err := ctr.WithExec([]string{"golangci-lint", "version"}).Sync(ctx); err != nil {
		tag, err := dag.Github().GetLatestRelease("golangci/golangci-lint").Tag(ctx)
		if err != nil {
			return "", err
		}

		// Install using the recommended approach: https://golangci-lint.run/welcome/install/
		cmd := []string{
			"curl",
			"-sSfL",
			"https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh",
			"|",
			"sh",
			"-s",
			"--",
			"-b",
			h.BinPath,
			tag,
		}
		ctr = ctr.WithExec([]string{"bash", "-c", strings.Join(cmd, " ")})
	}

	if format == "" {
		format = "colored-line-number"
	}

	cmd := []string{
		"golangci-lint",
		"run",
		"--timeout",
		"5m",
		"--out-format",
		format,
	}

	if h.Version != "latest" {
		cmd = append(cmd, "--go", h.Version)
	}

	return ctr.WithExec(cmd).Stdout(ctx)
}

// Format the source code within a target project using gofumpt. Formatted code must be
// copied back onto the host.`
func (h *Golang) Format(ctx context.Context) (*dagger.Directory, error) {
	ctr := h.Container
	if _, err := ctr.WithExec([]string{"gofumpt", "-version"}).Sync(ctx); err != nil {
		tag, err := dag.Github().GetLatestRelease("mvdan/gofumpt").Tag(ctx)
		if err != nil {
			return nil, err
		}

		ctr = ctr.WithExec([]string{"go", "install", "mvdan.cc/gofumpt@" + tag})
	}

	cmd := []string{"gofumpt", "-w", "-d", "."}

	return ctr.WithExec(cmd).Directory(goWorkDir), nil
}

// Test permit to run golang tests
func (h *Golang) Test(
	ctx context.Context,
	// if only short running tests should be executed
	// +optional
	short bool,
	// if the tests should be executed out of order
	// +optional
	shuffle bool,
	// run select tests only, defined using a regex
	// +optional
	run string,
	// skip select tests, defined using a regex
	// +optional
	skip string,
	// Run test with gotestsum
	// +optional
	withGotestsum bool,
	// Path to test
	// +optional
	path string,
	// The Kubeversion version to use
	// +optional
	// +default="latest"
	withKubeversion string,
) (result *TestResult, err error) {

	ctr := h.Container.
		WithExec(helper.ForgeCommand("go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest")).
		WithMountedCache("/tmp/envtest", dag.CacheVolume("envtest-k8s")).
		WithExec(helper.ForgeCommandf("setup-envtest use %s --bin-dir /tmp/envtest -p path", withKubeversion))

	stdout, err := ctr.Stdout(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when setup envtest")
	}

	ctr = ctr.WithEnvVariable("TEST", "true").
		WithEnvVariable("KUBEBUILDER_ASSETS", stdout)

	var cmd []string
	testPath := "./..."
	if path != "" {
		testPath = path
	}

	if withGotestsum {
		cmd = []string{"gotestsum", "--format", "testname", "--"}
		ctr = ctr.WithExec(helper.ForgeCommand("go install gotest.tools/gotestsum@latest"))
	} else {
		cmd = []string{"go", "test"}
	}
	cmd = append(cmd, "-p=1", "-count=1", "-vet=off", "-timeout=60m", "-covermode=atomic", "-coverprofile=coverage.out.tmp", testPath)

	if short {
		cmd = append(cmd, "-short")
	}

	if shuffle {
		cmd = append(cmd, "-shuffle=on")
	}

	if run != "" {
		cmd = append(cmd, []string{"-run", run}...)
	}

	if skip != "" {
		cmd = append(cmd, []string{"-skip", skip}...)
	}

	ctr = ctr.WithExec(cmd)

	return NewTestResult(ctr), nil
}

// WithSource permit to update the current source on sdk container
func (h *Golang) WithSource(
	// The source directory
	// +required
	src *dagger.Directory,
) *Golang {
	h.GolangModule.Container() = h.Container.WithDirectory(".", src)
	return h
}
