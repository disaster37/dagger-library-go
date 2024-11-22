package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/helper"
	"gopkg.in/yaml.v3"
)

type Sdk struct {
	Container *dagger.Container

	// +private
	BinPath string
}

func NewSdk(
	ctx context.Context,

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

) *Sdk {

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

	return &Sdk{
		Container: container.
			WithExec(helper.ForgeCommandf("curl --fail -L %s -o %s/operator-sdk", urlSdk, binPath)).
			WithExec(helper.ForgeCommandf("chmod +x %s/operator-sdk", binPath)).
			WithExec(helper.ForgeCommandf("curl --fail -L %s -o %s/opm", urlOpm, binPath)).
			WithExec(helper.ForgeCommandf("chmod +x %s/opm", binPath)).
			WithExec(helper.ForgeCommandf("go install %s", controllerGen)).
			WithExec(helper.ForgeCommandf("go install %s", cleanCrd)).
			WithExec(helper.ForgeCommandf("go install %s", kustomize)).
			WithExec(helper.ForgeCommand("go install github.com/mikefarah/yq/v4@latest")),
		BinPath: binPath,
	}
}

// Version display the current version of operator-sdk cli
func (h *Sdk) Version(
	ctx context.Context,
) (string, error) {
	return h.Container.WithExec(helper.ForgeCommand("operator-sdk version")).Stdout(ctx)
}

func (h *Sdk) Run(
	// The cmd to run on container
	// +required
	cmd string,
) *dagger.Container {

	return h.Container.WithExec(helper.ForgeCommand(cmd))
}

func (h *Sdk) Generate(
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
		crdSubCommand = fmt.Sprintf("crd:crdVersions=%s,generateEmbeddedObjectMeta=true", crdSubCommand)
	}

	return h.Container.
		WithExec(helper.ForgeCommandf("controller-gen rbac:roleName=%s %s webhook paths=\"./...\" output:crd:artifacts:config=config/crd/bases", roleName, crdSubCommand)).
		WithExec([]string{"crd", "clean-crd", "--crd-file", "config/crd/bases/*.yaml"}).
		WithExec(helper.ForgeCommand("controller-gen object:headerFile=\"hack/boilerplate.go.txt\" paths=\"./...\"")).
		Directory("."), nil
}

// Bundle generate the bundle
func (h *Sdk) Bundle(
	ctx context.Context,

	// The OCI image name without the version
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

// Prmit to run Kube with Operator
func (h *Sdk) Kube() *Kube {

	return NewKube(h.Container)

}

// WithSource permit to update the current source on sdk container
func (h *Sdk) WithSource(
	// The source directory
	// +required
	src *dagger.Directory,
) *Sdk {
	h.Container = h.Container.WithDirectory(".", src)
	return h
}
