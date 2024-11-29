package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"

	"github.com/disaster37/dagger-library-go/lib/helper"
)

type OperatorSdkGolang struct {
	// +private
	Src *dagger.Directory

	// +private
	*dagger.Golang
}

func NewGolang(
	// The source directory
	src *dagger.Directory,

	// The golang version to use when go.mod not exist
	// +optional
	version string,

	// Base container to use
	// +optional
	container *dagger.Container,

) *OperatorSdkGolang {
	return &OperatorSdkGolang{
		Src: src,
		Golang: dag.Golang(
			src,
			dagger.GolangOpts{
				Version: version,
				Base:    container,
			},
		),
	}
}

// Test permit to run golang tests
// It will run envtest with the kube version provided
func (h *OperatorSdkGolang) Test(
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
) *dagger.File {

	return h.Golang.
		With(func(r *dagger.Golang) *dagger.Golang {

			// Install and configure envtest
			ctr := h.Golang.Container().
				WithExec(helper.ForgeCommand("go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest")).
				WithMountedCache("/tmp/envtest", dag.CacheVolume("envtest-k8s")).
				WithExec(helper.ForgeCommandf("setup-envtest use %s --bin-dir /tmp/envtest -p path", withKubeversion))

			stdout, err := ctr.Stdout(ctx)
			if err != nil {
				panic(err)
			}

			ctr = ctr.WithEnvVariable("TEST", "true").
				WithEnvVariable("KUBEBUILDER_ASSETS", stdout)

			return dag.Golang(h.Src, dagger.GolangOpts{Base: ctr})
		}).
		Test(
			dagger.GolangTestOpts{
				Short:         short,
				Shuffle:       shuffle,
				Run:           run,
				Skip:          skip,
				WithGotestsum: withGotestsum,
				Path:          path,
			},
		)
}

// To update the source directory
func (h *OperatorSdkGolang) WithSource(
	// The source directory
	// +required
	src *dagger.Directory,
) *OperatorSdkGolang {
	h.Src = src
	h.Golang.WithSource(src)
	return h
}

// Container permit to get Golang container
func (h *OperatorSdkGolang) Container() *dagger.Container {
	return h.Golang.Container()
}
