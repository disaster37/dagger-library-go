package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"
	"encoding/json"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/helper"
	"golang.org/x/mod/modfile"
)

const (
	goMod     = "go.mod"
	goWorkDir = "/src"
)

type Golang struct {

	// +private
	Base *dagger.Container

	// +private
	Src *dagger.Directory

	// +private
	Version string
}

func NewGolang(
	ctx context.Context,

	// The source directory
	// +required
	src *dagger.Directory,

	// Container to use with operator-sdk cli inside and golang
	container *dagger.Container,
) *Golang {

	// Get the current golang version
	version, err := inspectModVersion(context.Background(), src)
	if err != nil {
		panic(err)
	}

	if container != nil {
		return &Golang{
			Base:    container,
			Src:     src,
			Version: version,
		}
	}

	// Compute the golang base container version
	base := defaultImage(version)
	base = mountCaches(ctx, base).
		WithDirectory(goWorkDir, src).
		WithWorkdir(goWorkDir).
		WithoutEntrypoint()

	return &Golang{
		Src:     src,
		Version: version,
		Base:    base,
	}
}

func (h *Golang) Sdk(
	ctx context.Context,

	// The operator-sdk cli version to use
	// +optional
	sdkVersion string,

	// The Opm cli version to use
	// +optional
	opmVersion string,

	// The controller gen version to use
	// +optional
	controllerGenVersion string,

	// The clean crd version to use
	// +optional
	cleanCrdVersion string,

	// The kustomize version to use
	// +optional
	kustomizeVersion string,

) *Sdk {
	return NewSdk(ctx, h.Base, sdkVersion, opmVersion, controllerGenVersion, cleanCrdVersion, kustomizeVersion)
}

func (h *Golang) Oci() *Oci {
	return NewOci(h.Base)
}

// Container return the container that contain golang
func (h *Golang) Container() *dagger.Container {
	return h.Base.WithDefaultTerminalCmd([]string{"bash"})
}

// Lint the target project using golangci-lint
func (h *Golang) Lint(
	ctx context.Context,
	// the type of report that should be generated
	// +optional
	// +default="colored-line-number"
	format string,
) (string, error) {
	ctr := h.Base
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
			"$(go env GOPATH)/bin",
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
	ctr := h.Base
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

	ctr := h.Base.
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
	h.Base = h.Base.WithDirectory(".", src)
	return h
}

func inspectModVersion(ctx context.Context, src *dagger.Directory) (string, error) {
	mod, err := src.File(goMod).Contents(ctx)
	if err != nil {
		// File not exist, use the last golang image
		return "latest", nil
	}

	f, err := modfile.Parse(goMod, []byte(mod), nil)
	if err != nil {
		return "", err
	}
	return f.Go.Version, nil
}

func defaultImage(version string) *dagger.Container {
	return dag.Container().From(fmt.Sprintf("golang:%s", version))
}

func mountCaches(ctx context.Context, base *dagger.Container) *dagger.Container {
	goEnvStdout, err := base.WithExec([]string{"go", "env", "-json"}).Stdout(ctx)
	if err != nil {
		panic(fmt.Sprintf("Error when get go env; %s", err.Error()))
	}
	var goEnv map[string]string
	if err := json.Unmarshal([]byte(goEnvStdout), &goEnv); err != nil {
		panic(fmt.Sprintf("Error when decode go env; %s", err.Error()))
	}

	goCacheEnv := goEnv["GOCACHE"]
	goModCacheEnv := goEnv["GOMODCACHE"]
	goBinCacheEnv := goEnv["GOBIN"]
	if goBinCacheEnv == "" {
		goBinCacheEnv = fmt.Sprintf("%s/bin", goEnv["GOPATH"])
	}

	gomod := dag.CacheVolume("gomod")
	gobuild := dag.CacheVolume("gobuild")
	gobin := dag.CacheVolume("gobin")

	return base.
		WithMountedCache(goModCacheEnv, gomod).
		WithMountedCache(goCacheEnv, gobuild).
		WithMountedCache(goBinCacheEnv, gobin)
}
