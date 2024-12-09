package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/helper"
	"gopkg.in/yaml.v3"
)

type OperatorSdkSdk struct {
	Container *dagger.Container

	// The source directory
	Src *dagger.Directory

	// +private
	BinPath string
}

func NewSdk(
	ctx context.Context,

	// The source directory
	// +required
	src *dagger.Directory,

	// Container to use with operator-sdk cli inside and golang
	// +required
	container *dagger.Container,

	// The bin path
	// +required
	binPath string,

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

) *OperatorSdkSdk {

	// Compute URL to download operator-sdk
	var urlSdk string
	if sdkVersion == "latest" || sdkVersion == "" {
		urlSdk = "https://github.com/operator-framework/operator-sdk/releases/latest/download/operator-sdk_linux_amd64"
	} else {
		urlSdk = fmt.Sprintf("https://github.com/operator-framework/operator-sdk/releases/download/%s/operator-sdk_linux_amd64", sdkVersion)
	}

	// Compute URL to download opm
	var urlOpm string
	if opmVersion == "latest" || opmVersion == "" {
		urlOpm = "https://github.com/operator-framework/operator-registry/releases/latest/download/linux-amd64-opm"
	} else {
		urlOpm = fmt.Sprintf("https://github.com/operator-framework/operator-registry/releases/download/%s/linux-amd64-opm", opmVersion)
	}

	// Compute the controllerGen version
	if controllerGenVersion == "latest" || controllerGenVersion == "" {
		controllerGenVersion = "latest"
	}
	controllerGen := fmt.Sprintf("sigs.k8s.io/controller-tools/cmd/controller-gen@%s", controllerGenVersion)

	// Compute the cleanCrd version to use
	if cleanCrdVersion == "latest" || cleanCrdVersion == "" {
		cleanCrdVersion = "latest"
	}
	cleanCrd := fmt.Sprintf("github.com/disaster37/operator-sdk-extra/cmd/crd@%s", cleanCrdVersion)

	// Compute kustomize to use
	if kustomizeVersion == "latest" || kustomizeVersion == "" {
		kustomizeVersion = "latest"
	}
	kustomize := fmt.Sprintf("sigs.k8s.io/kustomize/kustomize/v5@%s", kustomizeVersion)

	ctr := container.
		WithDirectory(".", src)

	// Install operator-sdk
	if _, err := ctr.WithExec([]string{"operator-sdk", "version"}).Sync(ctx); err != nil {
		ctr = ctr.WithExec(helper.ForgeCommandf("curl --fail -L %s -o %s/operator-sdk", urlSdk, binPath)).
			WithExec(helper.ForgeCommandf("chmod +x %s/operator-sdk", binPath))
	}

	// Install opm
	if _, err := ctr.WithExec([]string{"opm", "version"}).Sync(ctx); err != nil {
		ctr = ctr.WithExec(helper.ForgeCommandf("curl --fail -L %s -o %s/opm", urlOpm, binPath)).
			WithExec(helper.ForgeCommandf("chmod +x %s/opm", binPath))
	}

	// Install controller gen
	if _, err := ctr.WithExec([]string{"controller-gen", "--version"}).Sync(ctx); err != nil {
		ctr = ctr.WithExec(helper.ForgeCommandf("go install %s", controllerGen))
	}

	// Install clean crd
	if _, err := ctr.WithExec([]string{"crd", "--version"}).Sync(ctx); err != nil {
		ctr = ctr.WithExec(helper.ForgeCommandf("go install %s", cleanCrd))
	}

	// Install kustomize
	if _, err := ctr.WithExec([]string{"kustomize", "version"}).Sync(ctx); err != nil {
		ctr = ctr.WithExec(helper.ForgeCommandf("go install %s", kustomize))
	}

	// Install YQ
	if _, err := ctr.WithExec([]string{"yq", "--version"}).Sync(ctx); err != nil {
		ctr = ctr.WithExec(helper.ForgeCommand("go install github.com/mikefarah/yq/v4@latest"))
	}

	return &OperatorSdkSdk{
		Container: ctr,
		BinPath:   binPath,
		Src:       src,
	}
}

// Version display the current version of operator-sdk cli
func (h *OperatorSdkSdk) Version(
	ctx context.Context,
) (string, error) {
	return h.Container.WithExec(helper.ForgeCommand("operator-sdk version")).Stdout(ctx)
}

func (h *OperatorSdkSdk) Run(
	// The cmd to run on container
	// +required
	cmd string,
) *dagger.Container {

	return h.Container.WithExec(helper.ForgeCommand(cmd))
}

func (h *OperatorSdkSdk) GenerateManifests(
	ctx context.Context,

	// The CRD version to generate
	// +optional
	crdVersion string,
) (*dagger.Directory, error) {

	// Read the project file to get the project name. Use it to generate roleName to avoid colision with another operators
	pFile, err := h.Container.File("PROJECT").Contents(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when read file 'PROJECT'")
	}

	var data map[string]any

	if err = yaml.Unmarshal([]byte(pFile), &data); err != nil {
		return nil, errors.Wrap(err, "Error when decode 'PROJECT' file")
	}
	roleName := data["projectName"].(string)

	var crdSubCommand string
	if crdVersion == "" {
		crdSubCommand = "crd:generateEmbeddedObjectMeta=true"
	} else {
		crdSubCommand = fmt.Sprintf("crd:crdVersions=%s,generateEmbeddedObjectMeta=true", crdVersion)
	}

	return h.Container.
		WithExec([]string{"controller-gen", fmt.Sprintf("rbac:roleName=%s", roleName), crdSubCommand, "webhook", "paths=./...", "output:crd:artifacts:config=config/crd/bases"}).
		WithExec([]string{"crd", "clean-crd", "--crd-file", "config/crd/bases/*.yaml"}).
		WithExec([]string{"controller-gen", "object:headerFile=hack/boilerplate.go.txt", "paths=./..."}).
		Directory("."), nil
}

// Bundle generate the bundle
func (h *OperatorSdkSdk) GenerateBundle(
	ctx context.Context,

	// The OCI operator image name without the version
	// +required
	imageName string,

	// The current version
	// +required
	version string,

	// The channels
	// +optional
	channels string,

	// The previous version
	// +optional
	previousVersion string,
) (*dagger.Directory, error) {
	ctn := h.Container

	if previousVersion != "" {
		pFile, err := h.Container.File("PROJECT").Contents(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Error when read file 'PROJECT'")
		}

		var data map[string]any

		if err = yaml.Unmarshal([]byte(pFile), &data); err != nil {
			return nil, errors.Wrap(err, "Error when decode 'PROJECT' file")
		}
		projectName := data["projectName"].(string)
		ctn = ctn.WithExec([]string{
			"yq",
			"-i",
			fmt.Sprintf(".spec.replaces=\"%s.v%s\"", projectName, previousVersion),
			fmt.Sprintf("config/manifests/bases/%s.clusterserviceversion.yaml", projectName),
		})
	}

	var computeChannels string
	if channels != "" {
		computeChannels = fmt.Sprintf("--channels=%s", channels)
	}

	return ctn.WithExec(helper.ForgeCommand("operator-sdk generate kustomize manifests -q --apis-dir ./api")).
		WithExec(helper.ForgeScript("cd config/manager && kustomize edit set image controller=%s:%s", imageName, version)).
		WithExec(helper.ForgeScript("kustomize build config/manifests | operator-sdk generate bundle -q --overwrite --version %s %s", version, computeChannels)).
		WithExec(helper.ForgeCommand("operator-sdk bundle validate ./bundle")).
		Directory("."), nil

}

// WithSource permit to update the current source on sdk container
func (h *OperatorSdkSdk) WithSource(
	// The source directory
	// +required
	src *dagger.Directory,
) *OperatorSdkSdk {
	h.Container = h.Container.WithDirectory(".", src, dagger.ContainerWithDirectoryOpts{})
	h.Src = src
	return h
}

// InstallOlm permit to install the OLM
func (h *OperatorSdkSdk) InstallOlm(
	ctx context.Context,

	// The kubeconfig file to access on cluster where to install OLM
	// +required
	kubeconfig *dagger.File,

) (string, error) {

	return h.Container.
		WithFile("/kubeconfig", kubeconfig).
		WithEnvVariable("KUBECONFIG", "/kubeconfig").
		WithExec(helper.ForgeCommand("operator-sdk olm install")).
		Stdout(ctx)
}
