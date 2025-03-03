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
	kubeconfig *dagger.File,

	// Set true to install CRD prometheus.
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
	kubeCtr := h.Kube.Kube.Kubectl("version")

	if kubeconfig == nil {
		// Start kube cluster
		kubeService, err = h.Kube.KubeCluster(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Error when start K3s")
		}
		kubeconfig = h.Kube.Kube.Config()
		installPromteheusCrd = true

	} else {
		kubeCtr = kubeCtr.
			WithFile("/kubeconfig", kubeconfig).
			WithEnvVariable("KUBECONFIG", "/kubeconfig")
	}

	// Install OLM
	if _, err := h.Sdk.InstallOlm(
		ctx,
		kubeconfig,
	); err != nil {
		return nil, errors.Wrap(err, "Error when install OLM")
	}

	// Install Prometheus CRD
	if installPromteheusCrd {
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
		Stdout(ctx); err != nil {
		return nil, errors.Wrap(err, "Error when install subscription")
	}

	return kubeService, nil

}

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
) (*dagger.Service, error) {

	var service *dagger.Service
	var err error

	// Install OLM operator
	if service, err = h.InstallOlmOperator(
		ctx,
		catalogImage,
		name,
		channel,
		nil,
		true,
	); err != nil {
		return service, errors.Wrap(err, "Error when install OLM operator")
	}

	// Wait the time it install OLM operator
	kubeCtn := h.Kube.Kube.Kubectl("get nodes").
		WithExec([]string{"sleep", "120"})

	//  Get some trace to troobleshooting if needed
	_, _ = kubeCtn.
		WithExec(helper.ForgeCommand("kubectl -n olm get pods")).
		WithExec(helper.ForgeCommand("kubectl -n olm describe catalogSource test")).
		WithExec(helper.ForgeCommand("kubectl -n operators describe subscription test")).
		WithExec(helper.ForgeCommand("kubectl -n operators describe installplan")).
		WithExec(helper.ForgeCommand("kubectl -n operators describe clusterServiceVersion")).
		WithExec(helper.ForgeCommand("kubectl -n operators describe deployment")).
		WithExec(helper.ForgeCommand("kubectl -n operators describe pod")).
		WithExec(helper.ForgeScript("kubectl get -n operators pods -o name | xargs -I {} kubectl logs -n operators {}")).
		Stdout(ctx)

	// Check deployment operator is ready
	if _, err := kubeCtn.WithExec(helper.ForgeCommand("kubectl -n operators wait --for=condition=Available=True --all deployment --timeout=60s")).Stdout(ctx); err != nil {
		return service, errors.Wrap(err, "Operator not ready")
	}

	return service, nil

}

// RunOperator permit to run operator for test purpose
func (h *OperatorSdk) RunOperator(
	ctx context.Context,

	// The kubeconfig to connect on kube
	// If not set, it run local k3s
	// +optional
	kubeconfig *dagger.File,
) ([]*dagger.Service, error) {

	var err error
	var kubeService *dagger.Service
	kubeCtr := h.Kube.Kube.Kubectl("version")

	if kubeconfig == nil {
		// Start kube cluster
		kubeService, err = h.Kube.KubeCluster(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Error when start K3s")
		}
		kubeconfig = h.Kube.Kube.Config()

	} else {
		kubeCtr = kubeCtr.
			WithFile("/kubeconfig", kubeconfig).
			WithEnvVariable("KUBECONFIG", "/kubeconfig")
	}

	// Install CRD on kube
	crdFile := h.Sdk.Container.WithExec(helper.ForgeCommand("kustomize build config/crd -o /tmp/crd.yaml")).File("/tmp/crd.yaml")
	_, err = kubeCtr.
		WithFile("/tmp/crd.yaml", crdFile).
		WithExec(helper.ForgeCommand("kubectl apply --server-side=true -f /tmp/crd.yaml")).
		Stdout(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when install CRDs")
	}

	// Run operator as service
	operatorService, err := h.Golang.Container().
		WithFile("/tmp/kubeconfig", kubeconfig).
		WithEnvVariable("KUBECONFIG", "/tmp/kubeconfig").
		WithExposedPort(8081, dagger.ContainerWithExposedPortOpts{Protocol: dagger.NetworkProtocolTcp, Description: "Health"}).
		WithEnvVariable("ENABLE_WEBHOOKS", "false").
		WithEnvVariable("LOG_LEVEL", "trace").
		WithEnvVariable("LOG_FORMATTER", "json").
		WithEntrypoint(helper.ForgeCommand("go run cmd/main.go")).
		AsService().
		Start(ctx)

	if err != nil {
		return nil, errors.Wrap(err, "Error when run operator")
	}

	return []*dagger.Service{kubeService, operatorService}, nil

}

// Release permit to release to operator version
func (h *OperatorSdk) Release(
	ctx context.Context,

	// The version to release
	// +required
	version string,

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

) (*OperatorSdk, error) {

	var dir *dagger.Directory
	var err error

	if !skipBuildFromPreviousVersion && previousVersion == "" {
		// Open the current version
		previousVersion, _ = h.Src.File("VERSION").Contents(ctx)
	}
	if skipBuildFromPreviousVersion {
		previousVersion = ""
	}

	imageName := fmt.Sprintf("%s/%s", registry, repository)
	bundleName := fmt.Sprintf("%s-bundle", imageName)
	catalogName := h.GetCatalogName(registry, repository)
	fullImageName := fmt.Sprintf("%s:%s", imageName, version)
	fullBundleName := fmt.Sprintf("%s:%s", bundleName, version)
	fullCatalogName := fmt.Sprintf("%s:%s", catalogName, version)
	previousCatalogName := ""

	// Compute the previous catalog image
	if previousVersion != "" {
		previousCatalogName = fmt.Sprintf("%s:%s", catalogName, previousVersion)
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
	commit, _ := h.Oci.GolangContainer.
		WithExec(helper.ForgeCommand("git rev-parse --short HEAD")).
		Stdout(ctx)
	if _, err = h.Oci.
		BuildManager(ctx, version, commit, "").
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

	h.Src = dir

	return h, nil

}

func (h *OperatorSdk) GetSource() *dagger.Directory {
	return h.Src
}

// GetVersion permit to compute the target sem version
// Some time on CI, we should to build volatile version like PR or RC.
// When we are on this cas, we should to generate next minor version + tag
func (h *OperatorSdk) GetVersion(
	ctx context.Context,
	// The version to release
	// +required
	version string,

	// Set true if the current version is the build number
	// We will use semver from version file to generate next minor + version as tag name
	// +optional
	isBuildNumber bool,
) string {
	var nextVersion *semver.Version

	if isBuildNumber {
		// Open the current version
		previousVersionFromLocal, err := h.Src.File("VERSION").Contents(ctx)
		if err == nil {
			nextVersion = semver.New(previousVersionFromLocal)
		} else {
			nextVersion = semver.New("0.0.0")
		}

		if isBuildNumber {
			nextVersion.BumpPatch()
			nextVersion.Set(fmt.Sprintf("%s-%s", nextVersion.String(), version))
			version = nextVersion.String()
		}
	}

	return version
}

// GetCatalogName return the catalog image name
func (h *OperatorSdk) GetCatalogName(registry, repository string) string {
	return fmt.Sprintf("%s/%s-catalog", registry, repository)
}
