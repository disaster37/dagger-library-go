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

	// +private
	Base *dagger.Container
}

func NewSdk(
	ctx context.Context,

	// Container to use with operator-sdk cli inside and golang
	// + required
	container *dagger.Container,

	// The operator-sdk cli version to use
	// +optional
	sdkVersion string,

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
	var url string
	if sdkVersion == "latest" || sdkVersion == "" {
		url = "https://github.com/operator-framework/operator-sdk/releases/latest/download/operator-sdk_linux_amd64"
	} else {
		url = fmt.Sprintf("https://github.com/operator-framework/operator-sdk/releases/download/%s/operator-sdk_linux_amd64", sdkVersion)
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
		Base: container.
			WithExec(helper.ForgeCommandf("curl --fail -L %s -o /usr/bin/operator-sdk", url)).
			WithExec(helper.ForgeCommand("chmod +x /usr/bin/operator-sdk")).
			WithExec(helper.ForgeCommandf("go install %s", controllerGen)).
			WithExec(helper.ForgeCommandf("go install %s", cleanCrd)).
			WithExec((helper.ForgeCommandf("go install %s", kustomize))),
	}
}

// Container return the container that contain the operator-sdk cli
func (h *Sdk) Container() *dagger.Container {
	return h.Base.WithDefaultTerminalCmd([]string{"bash"})
}

// Version display the current version of operator-sdk cli
func (h *Sdk) Version(
	ctx context.Context,
) (string, error) {
	return h.Base.WithExec(helper.ForgeCommand("operator-sdk version")).Stdout(ctx)
}

func (h *Sdk) Run(
	// The cmd to run with operator-sdk
	// +required
	cmd string,
) *dagger.Directory {

	return h.Base.WithExec(helper.ForgeCommandf("operator-sdk %s", cmd)).Directory(".")
}

func (h *Sdk) Generate(
	ctx context.Context,

	// The CRD version to generate
	// +optional
	crdVersion string,
) (*dagger.Directory, error) {

	// Read the project file to get the project name. Use it to generate roleName to avoid colision with another operators
	pFile, err := h.Base.File("PROJECT").Contents(ctx)
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

	return h.Base.
		WithExec(helper.ForgeCommandf("controller-gen rbac:roleName=%s %s webhook paths=\"./...\" output:crd:artifacts:config=config/crd/bases", roleName, crdSubCommand)).
		WithExec([]string{"crd", "clean-crd", "--crd-file", "config/crd/bases/*.yaml"}).
		WithExec(helper.ForgeCommand("controller-gen object:headerFile=\"hack/boilerplate.go.txt\" paths=\"./...\"")).
		Directory("."), nil
}

// Bundle generate the bundle
func (h *Sdk) Bundle(
	ctx context.Context,

	// The OCI image name
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
	ctn := h.Base
	if previousVersion != "" {
		pFile, err := h.Base.File("PROJECT").Contents(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Error when read file 'PROJECT'")
		}

		var data map[string]any

		if err = yaml.Unmarshal([]byte(pFile), &data); err != nil {
			return nil, errors.Wrap(err, "Error when decode 'PROJECT' file")
		}
		projectName := data["projectName"].(string)
		ctn = ctn.WithExec(helper.ForgeCommandf("yq -yi '.spec.replaces=\"%s.v%s\"' config/manifests/bases/%s.clusterserviceversion.yaml", projectName, previousVersion, projectName))
	}

	var computeChannels string
	if channels != "" {
		computeChannels = fmt.Sprintf("--channels=%s", channels)
	}

	return ctn.WithExec(helper.ForgeCommand("operator-sdk generate kustomize manifests -q --apis-dir ./api")).
		WithExec(helper.ForgeScript("cd config/manager && kustomize edit set image controller=%s", imageName)).
		WithExec(helper.ForgeScript("kustomize build config/manifests | operator-sdk generate bundle -q --overwrite --version %s %s", version, computeChannels)).
		WithExec(helper.ForgeCommand("operator-sdk bundle validate ./bundle")).
		Directory("."), nil
}

func (h *Sdk) RunOnKube(
	ctx context.Context,

	// The kubeversion you should
	// +optional
	kubeVersion string,
) (*dagger.Container, error) {

	var err error
	k3s := dag.K3S("test")
	k3sServer := k3s.Server()
	k3sServer, err = k3sServer.Start(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when start k3s")
	}

	crdFile := h.Base.WithExec(helper.ForgeCommand("kustomize build config/crd -o /tmp/crd.yaml")).File("/tmp/crd.yaml")

	return k3s.Container().
		WithFile("/tmp/crd.yaml", crdFile).
		WithExec(helper.ForgeCommand("kubectl apply -f /tmp/crd.yaml")), nil

}

// WithSource permit to update the current source on sdk container
func (h *Sdk) WithSource(
	// The source directory
	// +required
	src *dagger.Directory,
) *Sdk {
	h.Base = h.Base.WithDirectory(".", src)
	return h
}