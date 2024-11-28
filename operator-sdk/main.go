// A generated module for OperatorSdk functions
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
	"bytes"
	"context"
	"dagger/operator-sdk/internal/dagger"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/helper"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/kubernetes/scheme"
)

type OperatorSdk struct {
	// +private
	Src *dagger.Directory

	// Docker module
	// +private
	Docker *dagger.DockerCli

	// K3s module
	// +private
	*Kube

	// The Golang module
	// +private
	*dagger.Golang

	// The SDK module
	// +private
	*Sdk

	// The OCI module
	// +private
	*Oci

	// +private
	KubeVersion string
}

func New(
	ctx context.Context,

	// The source directory
	// +required
	src *dagger.Directory,

	// Extra golang container
	// +optional
	container *dagger.Container,

	// The operator-sdk cli version to use
	// +optional
	sdkVersion string,

	// The opm cli version to use
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

	// The Docker version to use
	// +optional
	dockerVersion string,

	// The kube version to use when run tests
	kubeVersion string,

) (*OperatorSdk, error) {

	var err error

	// goModule
	goModule := dag.Golang(src, dagger.GolangOpts{Base: container})
	binPath, err := goModule.GoBin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when get GoBin")
	}

	//sdkModule
	sdkModule := NewSdk(
		ctx,
		src,
		goModule.Container(),
		binPath,
		sdkVersion,
		opmVersion,
		controllerGenVersion,
		cleanCrdVersion,
		kustomizeVersion,
	)

	// docker cli
	dockerCli := dag.Docker().Cli(
		dagger.DockerCliOpts{
			Version: dockerVersion,
		},
	)
	opmFile := sdkModule.Container.
		WithExec(helper.ForgeCommandf("cp %s/opm /tmp/opm", sdkModule.BinPath)).
		File("/tmp/opm")

	return &OperatorSdk{
		Src:    src,
		Golang: goModule,
		Sdk:    sdkModule,
		Oci: NewOci(
			src,
			goModule.Container(),
			dockerCli.Container().
				WithServiceBinding("dockerd.svc", dockerCli.Engine()).
				WithEnvVariable("DOCKER_HOST", "tcp://dockerd.svc:2375").
				WithFile("/usr/bin/opm", opmFile),
		),
		Docker:      dockerCli,
		Kube:        NewKube(src),
		KubeVersion: kubeVersion,
	}, nil
}

// WithSource permit to update source on all sub containers
func (h *OperatorSdk) WithSource(src *dagger.Directory) *OperatorSdk {
	h.Src = src
	h.Golang = h.Golang.WithSource(src)
	h.Sdk = h.Sdk.WithSource(src)
	h.Oci = h.Oci.WithSource(src)
	h.Kube = h.Kube.WithSource(src)

	return h
}

func (h *OperatorSdk) InstallOlmOperator(
	ctx context.Context,

	// The catalog image to install
	// +required
	catalogImage string,

	// The operator name
	// +required
	name string,

	// The channel of the operator to install
	// +optional
	// +default="stable"
	channel string,
) (*dagger.Service, error) {

	if channel == "" {
		channel = "stable"
	}

	// Start kube cluster
	kubeService, err := h.Kube.Kube.Server().Start(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when start K3s")
	}

	// Install OLM
	if _, err := h.Sdk.InstallOlm(
		ctx,
		h.Kube.Kube.Config(),
	); err != nil {
		return nil, errors.Wrap(err, "Error when install OLM")
	}

	// Forge Catalog
	catalogSource := &olmv1alpha1.CatalogSource{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test",
			Namespace: "olm",
		},
		Spec: olmv1alpha1.CatalogSourceSpec{
			SourceType: olmv1alpha1.SourceTypeGrpc,
			Image:      catalogImage,
		},
	}

	sch := scheme.Scheme
	if err := olmv1alpha1.AddToScheme(sch); err != nil {
		panic(err)
	}
	y := printers.NewTypeSetter(sch).ToPrinter(&printers.YAMLPrinter{})
	buf := new(bytes.Buffer)
	if err := y.PrintObj(catalogSource, buf); err != nil {
		panic(err)
	}

	// Install catalog
	if _, err := h.Kube.Kube.Kubectl("version").
		WithNewFile("/tmp/catalog.yaml", buf.String()).
		Terminal().
		WithExec(helper.ForgeCommand("kubectl apply --server-side=true -f /tmp/catalog.yaml")).
		Stdout(ctx); err != nil {
		return nil, errors.Wrap(err, "Error when install catalog")
	}

	// Forge subscription
	subscription := &olmv1alpha1.Subscription{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test",
			Namespace: "olm",
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			CatalogSource:          "test",
			CatalogSourceNamespace: "olm",
			Channel:                channel,
			InstallPlanApproval:    olmv1alpha1.ApprovalAutomatic,
			Package:                name,
		},
	}
	y = printers.NewTypeSetter(sch).ToPrinter(&printers.YAMLPrinter{})
	buf = new(bytes.Buffer)
	if err := y.PrintObj(subscription, buf); err != nil {
		panic(err)
	}

	// Install subscription
	if _, err := h.Kube.Kube.Kubectl("version").
		WithNewFile("/tmp/subscription.yaml", buf.String()).
		WithExec(helper.ForgeCommand("kubectl apply --server-side=true -f  /tmp/subscription.yaml")).
		Stdout(ctx); err != nil {
		return nil, errors.Wrap(err, "Error when install subscription")
	}

	return kubeService, nil

}

/*

// Release permit to release to operator version
func (h *OperatorSdk) Release(
	ctx context.Context,

	// The version to release
	// +required
	version string,

	// The previous version to replace
	// +optional
	previousVersion string,

	// The CRD version do generate manifests
	// +optional
	crdVersion string,

	// The list of channel. Comma separated
	// +optional
	channels string,

	// Set true to run tests
	// +optional
	withTest bool,

	// The OCI registry
	// +required
	registry string,

	// The OCI repository
	// +required
	repository string,

	// The registry username
	// +required
	registryUsername string,

	// The registry password
	// +required
	registryPassword *dagger.Secret,

) (*dagger.Directory, error) {

	var dir *dagger.Directory
	var err error

	// Prepare OCI
	h.Oci.WithRepositoryCredentials(registry, registryUsername, registryPassword)

	// Generate manifests
	dir, err = h.Generate(ctx, crdVersion)
	if err != nil {
		return nil, errors.Wrap(err, "Error when call 'generate'")
	}
	h.WithSource(dir)

	// Format code
	dir = h.Format()
	h.WithSource(dir)

	// Lint code
	_, err = h.Lint(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when call 'lint'")
	}

	if withTest {
		coverageFile := h.Test(ctx, false, false, "", "", true, "", h.KubeVersion)
		dir = dir.WithFile(".", coverageFile)
		h.WithSource(dir)
	}

	// Run bundle
	metadata := &metadata{}
	versionFile, err := h.Sdk.Container.File("version.yaml").Contents(ctx)
	if err == nil {
		if err := yaml.Unmarshal([]byte(versionFile), metadata); err != nil {
			return nil, errors.Wrap(err, "Error when decode version.yaml")
		}
	}
	if previousVersion == "" {
		metadata.PreviousVersion = metadata.CurrentVersion
	} else {
		metadata.PreviousVersion = previousVersion
	}
	metadata.CurrentVersion = version
	dir, err = h.Sdk.Bundle(
		ctx,
		fmt.Sprintf("%s/%s", registry, repository),
		metadata.CurrentVersion,
		channels,
		metadata.PreviousVersion,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error when call 'bundle'")
	}
	h.WithSource(dir)

	// Build and push operator image
	_, err = h.PublishManager(ctx, fmt.Sprintf("%s/%s:%s", registry, repository, metadata.CurrentVersion))
	if err != nil {
		return nil, errors.Wrap(err, "Error when call 'publishManager'")
	}

	// Build and push the bundle
	_, err = h.PublishBundle(ctx, fmt.Sprintf("%s/%s-bundle:v%s", registry, repository, metadata.CurrentVersion))
	if err != nil {
		return nil, errors.Wrap(err, "Error when call 'publishBundle'")
	}

	// Build and push catalog
	updateFromPreviousCatalog := true
	if metadata.CurrentVersion == "0.0.1" {
		updateFromPreviousCatalog = false
	}
	h.BuildCatalog(
		ctx,
		fmt.Sprintf("%s/%s-catalog:latest", registry, repository),
		fmt.Sprintf("%s/%s-catalog:%s", registry, repository, metadata.CurrentVersion),
		fmt.Sprintf("%s/%s-bundle:v%s", registry, repository, metadata.CurrentVersion),
		updateFromPreviousCatalog,
	)

	// @TODO write the new version file

	return dir, nil

}

type metadata struct {
	CurrentVersion  string `yaml:"currentVersion"`
	PreviousVersion string `yaml:"previousVersion"`
}

*/
