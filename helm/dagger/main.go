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
	"github.com/disaster37/dagger-library-go/helm/dagger/internal/dagger"
	"github.com/disaster37/dagger-library-go/helper"
)

type Helm struct {
	baseHelmContainer      *dagger.Container
	baseGeneratorContainer *dagger.Container
	baseYqContainer        *dagger.Container
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
		helm.baseHelmContainer = baseHelmContainer
	} else {
		helm.baseHelmContainer = helm.BaseHelmContainer()
	}

	if baseGeneratorContainer != nil {
		helm.baseGeneratorContainer = baseGeneratorContainer
	} else {
		helm.baseGeneratorContainer = helm.BaseGeneratorContainer()
	}

	if baseYqContainer != nil {
		helm.baseYqContainer = baseYqContainer
	} else {
		helm.baseYqContainer = helm.BaseYqContainer()
	}

	return helm
}

// BaseGeneratorContainer return the default image for readme-generator-for-helm
func (m *Helm) BaseGeneratorContainer() *dagger.Container {
	return dag.Container().
		From("node:21-alpine").
		WithExec(helper.ForgeCommand("npm install -g @bitnami/readme-generator-for-helm"))
}

// BaseHelmContainer return the default image for helm
func (m *Helm) BaseHelmContainer() *dagger.Container {
	return dag.Container().
		From("alpine/helm:3.14.3")
}

// BaseYqContainer return the default image for yq
func (m *Helm) BaseYqContainer() *dagger.Container {
	return dag.Container().
		From("mikefarah/yq:4.35.2")
}
