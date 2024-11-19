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
	"context"
	"dagger/operator-sdk/internal/dagger"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"gopkg.in/yaml.v3"
)

type OperatorSdk struct {
	// +private
	Src *dagger.Directory
}

func New(
	ctx context.Context,

	// The source directory
	src *dagger.Directory,

) *OperatorSdk {
	return &OperatorSdk{
		Src: src,
	}
}

func (h *OperatorSdk) Golang(
	ctx context.Context,

	// Set alternative Golang container
	// +optional
	container *dagger.Container,
) *Golang {
	return NewGolang(ctx, h.Src, container)
}

func (h *OperatorSdk) Release(
	ctx context.Context,

	// The list of channel. Comma separated
	// +optional
	channels string,

	// Set true to run tests
	// +optional
	// +default=false
	withTest bool,

	// Set alternative Golang container
	// +optional
	container *dagger.Container,

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

	// The CRD version to generate
	// +optional
	crdVersion string,

	// The Kubeversion version to use when run test
	// +optional
	// +default="latest"
	withKubeversion string,

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

	var sb strings.Builder
	var dir *dagger.Directory
	var err error
	var stdout string

	golang := h.Golang(ctx, container)
	sdk := golang.Sdk(ctx, sdkVersion, opmVersion, controllerGenVersion, cleanCrdVersion, kustomizeVersion)

	// Generate manifests
	dir, err = sdk.Generate(ctx, crdVersion)
	if err != nil {
		return nil, errors.Wrap(err, "Error when call 'generate'")
	}
	golang = golang.WithSource(dir)
	sdk = sdk.WithSource(dir)

	// Format code
	dir, err = golang.Format(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when call 'format'")
	}
	golang = golang.WithSource(dir)
	sdk = sdk.WithSource(dir)

	// Lint code
	stdout, err = golang.Lint(ctx, "")
	if err != nil {
		return nil, errors.Wrap(err, "Error when call 'lint'")
	}
	sb.WriteString(stdout)
	sb.WriteString("\n")

	if withTest {
		res, err := golang.Test(ctx, false, false, "", "", true, "", withKubeversion)
		if err != nil {
			return nil, errors.Wrap(err, "Error when call 'test'")
		}
		stdout, err = res.Stdout(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Error when run tests")
		}
		sb.WriteString(stdout)
		sb.WriteString("\n")
		dir = dir.WithFile(".", res.Coverage())
		golang = golang.WithSource(dir)
		sdk = sdk.WithSource(dir)
	}

	// Run bundle
	metadata := &metadata{}
	versionFile, err := sdk.Container.File("version.yaml").Contents(ctx)
	if err != nil {
		// File not yet exist
		metadata.CurrentVersion = "0.0.1"
	} else {
		if err := yaml.Unmarshal([]byte(versionFile), metadata); err != nil {
			return nil, errors.Wrap(err, "Error when decode version.yaml")
		}

		if metadata.CurrentVersion == "" {
			return nil, errors.New("Your file version.yaml have not field 'currentVersion'")
		}

		if metadata.PreviousVersion == "" {
			return nil, errors.New("Your file version.yaml have not field 'previousVersion'")
		}
	}

	dir, err = sdk.Bundle(ctx, fmt.Sprintf("%s/%s", registry, repository), metadata.CurrentVersion, channels, metadata.PreviousVersion)
	if err != nil {
		return nil, errors.Wrap(err, "Error when call 'bundle'")
	}
	golang = golang.WithSource(dir)
	sdk = sdk.WithSource(dir)

	oci := golang.Oci().WithRepositoryCredentials(registry, registryUsername, registryPassword)

	// Build and push operator image
	stdout, err = oci.PublishManager(ctx, fmt.Sprintf("%s/%s:%s", registry, repository, metadata.CurrentVersion))
	if err != nil {
		return nil, errors.Wrap(err, "Error when call 'publishManager'")
	}
	sb.WriteString(stdout)
	sb.WriteString("\n")

	// Build and push the bundle
	stdout, err = oci.PublishBundle(ctx, fmt.Sprintf("%s/%s-bundle:v%s", registry, repository, metadata.CurrentVersion))
	if err != nil {
		return nil, errors.Wrap(err, "Error when call 'publishBundle'")
	}
	sb.WriteString(stdout)
	sb.WriteString("\n")

	// @TODO write the new version file

	return dir, nil

}

type metadata struct {
	CurrentVersion  string `yaml:"currentVersion"`
	PreviousVersion string `yaml:"previousVersion"`
}
