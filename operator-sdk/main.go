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
	"fmt"

	"emperror.dev/errors"
	"github.com/coreos/go-semver/semver"
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
	Kube *OperatorSdkKube

	// The Golang module
	Golang *OperatorSdkGolang

	// The SDK module
	Sdk *OperatorSdkSdk

	// The OCI module
	Oci *OperatorSdkOci
}

func New(
	ctx context.Context,

	// The source directory
	// +required
	src *dagger.Directory,

	// Extra golang container
	// +optional
	container *dagger.Container,

	// The go version when go.mod not yet exist
	// +optional
	goVersion string,

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

) (*OperatorSdk, error) {

	var err error

	// goModule
	goModule := NewGolang(src, goVersion, container)
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
		Docker: dockerCli,
		Kube:   NewKube(src),
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

	// The kubeconfig to connect on existing cluster
	// It not set, it will run local k3s cluster
	// +optional
	kubeconfig string,

	// Set tru to install CRD prometheus.
	// When you use internal kube, it always true
	// The installPlan needed this if metric is enable on operator
	// +optional
	installPromteheusCrd bool,
) (*dagger.Service, error) {

	if channel == "" {
		channel = "stable"
	}

	var err error
	var kubeService *dagger.Service
	var kubeConfigFile *dagger.File
	kubeCtr := h.Kube.Kube.Kubectl("version")

	if kubeconfig == "" {
		// Start kube cluster
		kubeService, err = h.Kube.Kube.Server().Start(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Error when start K3s")
		}
		kubeConfigFile = h.Kube.Kube.Config()
	} else {
		kubeConfigFile = dag.Directory().WithNewFile("kubeconfig", kubeconfig).File("kubeconfig")
		kubeCtr = kubeCtr.
			WithFile("/kubeconfig", kubeConfigFile).
			WithEnvVariable("KUBECONFIG", "/kubeconfig")
	}

	// Install OLM
	if _, err := h.Sdk.InstallOlm(
		ctx,
		kubeConfigFile,
	); err != nil {
		return nil, errors.Wrap(err, "Error when install OLM")
	}

	// Install Prometheus CRD
	if kubeconfig == "" || installPromteheusCrd {
		if _, err := kubeCtr.
			WithExec(helper.ForgeCommand("kubectl apply --server-side=true -f https://raw.githubusercontent.com/prometheus-community/helm-charts/refs/heads/main/charts/kube-prometheus-stack/charts/crds/crds/crd-servicemonitors.yaml")).
			WithExec(helper.ForgeCommand("kubectl apply --server-side=true -f https://raw.githubusercontent.com/prometheus-community/helm-charts/refs/heads/main/charts/kube-prometheus-stack/charts/crds/crds/crd-podmonitors.yaml")).
			Stdout(ctx); err != nil {
			return nil, errors.Wrap(err, "Error when install ServiceMonitor / PodMonitor CRD")
		}
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
	if _, err := kubeCtr.
		WithNewFile("/tmp/catalog.yaml", buf.String()).
		WithExec(helper.ForgeCommand("kubectl apply --server-side=true -f /tmp/catalog.yaml")).
		//WithExec(helper.ForgeCommand("kubectl wait catalogSource test --for=jsonpath=status.connectionState.lastObservedState=READY -n olm --timeout 60s")).
		Stdout(ctx); err != nil {
		return nil, errors.Wrap(err, "Error when install catalog")
	}

	// Forge subscription
	subscription := &olmv1alpha1.Subscription{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test",
			Namespace: "operators",
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
	if _, err := kubeCtr.
		WithNewFile("/tmp/subscription.yaml", buf.String()).
		WithExec(helper.ForgeCommand("kubectl apply --server-side=true -f /tmp/subscription.yaml")).
		//WithExec(helper.ForgeCommand("kubectl wait subscription test --for=jsonpath=status.state=AtLatestKnown -n operators --timeout 60s")).
		Stdout(ctx); err != nil {
		return nil, errors.Wrap(err, "Error when install subscription")
	}

	return kubeService, nil

}

/*
// It will deploy OLM, Then it will deploy operator on it
// Then it will check that the operator pod run
func (h *OperatorSdk) TestOlmOperator(
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
) (string, error) {

	// Install OLM operator
	if _, err := h.InstallOlmOperator(
		ctx,
		catalogImage,
		name,
		channel,
		"",
		true,
	); err != nil {
		return "", errors.Wrap(err, "Error when install OLM operator")
	}

	if _, err := h.Kube.Kube.Kubectl(fmt.Sprintf("-n operators get pod")).Stdout(ctx) {

	}
}
*/

// Release permit to release to operator version
func (h *OperatorSdk) Release(
	ctx context.Context,

	// The version to release
	// +required
	version string,

	// Set true if the current version is the build number
	// We will use semver from version file to generate next minor + version as tag name
	// +optional
	isBuildNumber bool,

	// The previous version to replace
	// +optional
	previousVersion string,

	// Set tru to not build from previous version
	// It usefull when build from PR
	// +optional
	skipBuildFromPreviousVersion bool,

	// The CRD version do generate manifests
	// +optional
	crdVersion string,

	// The list of channel. Comma separated
	// +optional
	channels string,

	// Set true to run tests
	// +optional
	withTest bool,

	// Set the kubeversion to use when run envtest
	// +optional
	kubeVersion string,

	// Set true to publish the operator image, the bundle image and the catalog image
	// +optional
	withPublish bool,

	// Set true to publish the catalog with last tag
	// +optional
	publishLast bool,

	// The OCI registry
	registry string,

	// The OCI repository
	repository string,

	// The registry username
	// +optional
	registryUsername string,

	// The registry password
	// +optional
	registryPassword *dagger.Secret,

) (*dagger.Directory, error) {

	var dir *dagger.Directory
	var err error
	var previousVersionFromLocal string
	var nextVersion *semver.Version

	if isBuildNumber || (!skipBuildFromPreviousVersion && (previousVersion == "")) {
		// Open the current version
		previousVersionFromLocal, err := h.Src.File("VERSION").Contents(ctx)
		if err == nil {
			nextVersion = semver.New(previousVersionFromLocal)
		} else {
			nextVersion = semver.New("0.0.0")
		}
	}

	imageName := fmt.Sprintf("%s/%s", registry, repository)
	bundleName := fmt.Sprintf("%s-bundle", imageName)
	catalogName := fmt.Sprintf("%s-catalog", imageName)
	fullImageName := fmt.Sprintf("%s:%s", imageName, version)
	fullBundleName := fmt.Sprintf("%s:%s", bundleName, version)
	fullCatalogName := fmt.Sprintf("%s:%s", catalogName, version)
	previousCatalogName := ""

	

	if isBuildNumber || (!skipBuildFromPreviousVersion && (previousVersion == "")) {
		// Open the current version
		previousVersionFromLocal, err := h.Src.File("VERSION").Contents(ctx)
		if err == nil {
			nextVersion = semver.New(previousVersionFromLocal)
		} else {
			nextVersion = semver.New("0.0.0")
		}
	}

	if isBuildNumber {
		nextVersion.BumpPatch()
		nextVersion.Set(fmt.Sprintf("%s-%s", nextVersion.String(), version))
		version = nextVersion.String()
	}

	// Compute the last version
	if skipBuildFromPreviousVersion {
		previousVersion = ""
	} else {
		if previousVersion != "" {
			previousCatalogName = fmt.Sprintf("%s:%s", catalogName, previousVersion)
		} else {
			// Open the current version
			if previousVersionFromLocal != "" {
				previousCatalogName = fmt.Sprintf("%s:%s", catalogName, previousVersionFromLocal)
			}
		}
	}

	lastCatalogName := fmt.Sprintf("%s:latest", catalogName)

	// Generate manifests
	dir, err = h.Sdk.GenerateManifests(ctx, crdVersion)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate manifests")
	}
	h = h.WithSource(dir)

	// Generate bundle
	dir, err = h.Sdk.GenerateBundle(
		ctx,
		imageName,
		version,
		channels,
		previousVersion,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate bundle")
	}
	h = h.WithSource(dir)

	// Format code
	dir = h.Golang.Golang.Format()
	h = h.WithSource(dir)

	// Lint code
	if _, err = h.Golang.Golang.Lint(ctx); err != nil {
		return nil, errors.Wrap(err, "Error when lint code")
	}

	// Vuln check
	if _, err = h.Golang.Golang.Vulncheck(ctx); err != nil {
		return nil, errors.Wrap(err, "Error when check vulnerability")
	}

	// Test code
	if withTest {
		coverageFile := h.Golang.Test(
			ctx,
			false,
			false,
			"",
			"",
			true,
			"",
			kubeVersion,
		)
		dir = dir.WithFile("coverage.out", coverageFile)
		h = h.WithSource(dir)
	}

	// Build operator image
	if _, err = h.Oci.
		BuildManager(ctx).
		Manager.
		Sync(ctx); err != nil {
		return nil, errors.Wrap(err, "Error when build operator image")
	}

	// Build bundle
	if _, err = h.Oci.
		BuildBundle(ctx).
		Bundle.
		Sync(ctx); err != nil {
		return nil, errors.Wrap(err, "Error when build bundle image")
	}

	if withPublish {
		if registryUsername == "" || registryPassword == nil {
			return nil, errors.New("You need to provide credentials to connect on registry to publish images")
		}

		// Add registry credentials
		h.Oci.WithRepositoryCredentials(registry, registryUsername, registryPassword)

		// Publish operator image
		if _, err := h.Oci.PublishManager(ctx, fullImageName); err != nil {
			return nil, errors.Wrap(err, "Error when Publish operator image")
		}
		fmt.Printf("Successfully publish operator image: %s\n", fullImageName)

		// Publish bundle image
		if _, err := h.Oci.PublishBundle(ctx, fullBundleName); err != nil {
			return nil, errors.Wrap(err, "Error when publish bundle image")
		}
		fmt.Printf("Successfully publish bundle image: %s\n", fullBundleName)

		// Build catalog
		// We can only build catalog after publish the bundle
		if _, err = h.Oci.BuildCatalog(
			ctx,
			fullCatalogName,
			previousCatalogName,
			fullBundleName,
		); err != nil {
			return nil, errors.Wrap(err, "Error when build Catalog image")
		}

		// Publish catalog image
		if _, err := h.Oci.PublishCatalog(ctx, fullCatalogName); err != nil {
			return nil, errors.Wrap(err, "Error when publish catalog image")
		}

		fmt.Printf("Successfully publish catalog image: %s\n", fullCatalogName)

		if publishLast {
			if _, err := h.Oci.PublishCatalog(ctx, lastCatalogName); err != nil {
				return nil, errors.Wrap(err, "Error when publish last catalog image")
			}

			fmt.Printf("Successfully publish catalog image: %s\n", lastCatalogName)
		}

	}

	// Generate current version file
	dir = dir.WithNewFile("VERSION", version)

	return dir, nil

}
