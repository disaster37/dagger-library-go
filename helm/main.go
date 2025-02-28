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

type Helm struct {
	BaseHelmContainer      *dagger.Container
	BaseGeneratorContainer *dagger.Container
	BaseYqContainer        *dagger.Container
}

func New(
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
	helm := &Helm{}

	if baseHelmContainer != nil {
		helm.BaseHelmContainer = baseHelmContainer
	} else {
		helm.BaseHelmContainer = helm.GetBaseHelmContainer()
	}

	if baseGeneratorContainer != nil {
		helm.BaseGeneratorContainer = baseGeneratorContainer
	} else {
		helm.BaseGeneratorContainer = helm.GetBaseGeneratorContainer()
	}

	if baseYqContainer != nil {
		helm.BaseYqContainer = baseYqContainer
	} else {
		helm.BaseYqContainer = helm.GetBaseYqContainer()
	}

	return helm
}

// BaseGeneratorContainer return the default image for readme-generator-for-helm
func (m *Helm) GetBaseGeneratorContainer() *dagger.Container {
	return dag.Container().
		From("node:21-alpine").
		WithExec(helper.ForgeCommand("npm install -g @bitnami/readme-generator-for-helm"))
}

// BaseHelmContainer return the default image for helm
func (m *Helm) GetBaseHelmContainer() *dagger.Container {
	return dag.Container().
		From("alpine/helm:3.14.3")
}

// BaseYqContainer return the default image for yq
func (m *Helm) GetBaseYqContainer() *dagger.Container {
	return dag.Container().
		From("mikefarah/yq:4.35.2")
}

// WithRepository permit to login on private helm repository
func (m *Helm) WithRepository(
	ctx context.Context,

	// The repository name
	name string,

	// The repository url
	url string,

	// Is it an OCI repository
	// +default=false
	isOci bool,

	// The repository username
	// +optional
	username *dagger.Secret,

	// The repository password
	// +optional
	password *dagger.Secret,

) *Helm {

	re := regexp.MustCompile(`(-|/)`)

	usernameEnv := fmt.Sprintf("REGISTRY_USERNAME_%s", strings.ToUpper(re.ReplaceAllString(name, "_")))
	passwordEnv := fmt.Sprintf("REGISTRY_PASSWORD_%s", strings.ToUpper(re.ReplaceAllString(name, "_")))
	m.BaseHelmContainer = m.BaseHelmContainer.
		WithSecretVariable(usernameEnv, username).
		WithSecretVariable(passwordEnv, password)
	if isOci {
		m.BaseHelmContainer = m.BaseHelmContainer.WithExec(helper.ForgeScript("helm registry login -u ${%s} -p ${%s} %s", usernameEnv, passwordEnv, url))
	} else {
		m.BaseHelmContainer = m.BaseHelmContainer.WithExec(helper.ForgeScript("helm repo add --username ${%s} --password ${%s} %s %s", usernameEnv, passwordEnv, name, url))
	}

	return m
}
