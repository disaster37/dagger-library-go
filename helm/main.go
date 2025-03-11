// A generated module for Helm functions
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
	"dagger/helm/internal/dagger"
	"fmt"
	"regexp"
	"strings"

	"github.com/disaster37/dagger-library-go/lib/helper"
)

//go:generate go get -u github.com/valyala/quicktemplate/qtc
//go:generate qtc -dir=templates

const sourceDirectory = "/source"

type Helm struct {
	// +private
	Src                *dagger.Directory
	HelmContainer      *dagger.Container
	GeneratorContainer *dagger.Container
	YqContainer        *dagger.Container
}

func New(
	// The helm source
	// +required
	src *dagger.Directory,

	// base helm container
	// It need contain helm
	// +optional
	baseHelmContainer *dagger.Container,

	// Base generator container
	// It need contain readme-generator-for-helm
	// +optional
	baseGeneratorContainer *dagger.Container,

	// Base YQ container
	// It need contain yq
	// +optional
	baseYqContainer *dagger.Container,
) *Helm {
	helm := &Helm{
		Src: src,
	}

	if baseHelmContainer != nil {
		helm.HelmContainer = baseHelmContainer
	} else {
		helm.HelmContainer = dag.Container().From("alpine/helm:3.14.3")
	}
	helm.HelmContainer = helm.HelmContainer.WithWorkdir(sourceDirectory)

	if baseGeneratorContainer != nil {
		helm.GeneratorContainer = baseGeneratorContainer
	} else {
		helm.GeneratorContainer = dag.Container().
			From("node:21-alpine").
			WithExec(helper.ForgeCommand("npm install -g @bitnami/readme-generator-for-helm"))
	}
	helm.GeneratorContainer = helm.GeneratorContainer.WithWorkdir(sourceDirectory)

	if baseYqContainer != nil {
		helm.YqContainer = baseYqContainer
	} else {
		helm.YqContainer = dag.Container().From("mikefarah/yq:4.35.2")
	}
	helm.YqContainer = helm.YqContainer.WithWorkdir(sourceDirectory)

	helm = helm.WithSource(src)

	return helm
}

// WithRepository permit to login on private helm repository
func (m *Helm) WithRepository(
	ctx context.Context,

	// The repository name
	// You need to set it when is not OCI format
	// +optional
	name string,

	// The repository url
	url string,

	// Is it an OCI repository
	// +optional
	// +default=false
	isOci bool,

	// The repository username
	// +required
	username *dagger.Secret,

	// The repository password
	// +required
	password *dagger.Secret,

) *Helm {

	if !isOci && name == "" {
		panic("You need to provide name when is not OCI registry")
	}

	re := regexp.MustCompile(`(-|/)`)

	usernameEnv := fmt.Sprintf("REGISTRY_USERNAME_%s", strings.ToUpper(re.ReplaceAllString(name, "_")))
	passwordEnv := fmt.Sprintf("REGISTRY_PASSWORD_%s", strings.ToUpper(re.ReplaceAllString(name, "_")))
	m.HelmContainer = m.HelmContainer.
		WithSecretVariable(usernameEnv, username).
		WithSecretVariable(passwordEnv, password)
	if isOci {
		m.HelmContainer = m.HelmContainer.WithExec(helper.ForgeScript("helm registry login -u ${%s} -p ${%s} %s", usernameEnv, passwordEnv, url))
	} else {
		m.HelmContainer = m.HelmContainer.WithExec(helper.ForgeScript("helm repo add --username ${%s} --password ${%s} %s %s", usernameEnv, passwordEnv, name, url))
	}

	return m
}

// WithSource permit to update the current source
func (h *Helm) WithSource(
	// The source directory
	// +required
	src *dagger.Directory,
) *Helm {
	h.Src = src

	// Add source
	h.HelmContainer = h.HelmContainer.WithDirectory(".", src)
	h.GeneratorContainer = h.GeneratorContainer.WithDirectory(".", src)
	h.YqContainer = h.YqContainer.WithDirectory(".", src)

	return h
}

// WithWorkDir change the working directory on all containers
func (h *Helm) WithWorkDir(
	// The path where to go as working directory
	// +required
	path string,
) *Helm {

	h.Src = h.Src.Directory(path)

	// Add source
	h.HelmContainer = h.HelmContainer.WithWorkdir(path)
	h.GeneratorContainer = h.GeneratorContainer.WithWorkdir(path)
	h.YqContainer = h.YqContainer.WithWorkdir(path)

	return h
}
