package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/helper"
)

const (
	goMod     = "go.mod"
	goWorkDir = "/src"
)

type Golang struct {
	*dagger.Golang
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
		Golang: dag.Golang(src, dagger.GolangOpts{Base: container}),
	}
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

	// Add axtra tools ton Golang container
	ctr := h.Container().
		WithExec(helper.ForgeCommand("go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest")).
		WithMountedCache("/tmp/envtest", dag.CacheVolume("envtest-k8s")).
		WithExec(helper.ForgeCommandf("setup-envtest use %s --bin-dir /tmp/envtest -p path", withKubeversion))

	stdout, err := ctr.Stdout(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when setup envtest")
	}

	ctr = ctr.WithEnvVariable("TEST", "true").
		WithEnvVariable("KUBEBUILDER_ASSETS", stdout)

		// Create new Golang module with our extra container to run tests
	golangModule := dag.Golang(h.Container().Directory("."), dagger.GolangOpts{Base: ctr})

	res := golangModule.Test(
		dagger.GolangTestOpts{
			Short:         short,
			Shuffle:       shuffle,
			Run:           run,
			Skip:          skip,
			WithGotestsum: withGotestsum,
			Path:          path,
		},
	)

	return NewTestResult(res), nil
}

type TestResult struct {
	*dagger.GolangTest
}

func NewTestResult(res *dagger.GolangTest) *TestResult {
	return &TestResult{
		GolangTest: res,
	}
}
