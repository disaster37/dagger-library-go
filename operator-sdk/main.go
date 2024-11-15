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
)

type OperatorSdk struct {
	// +private
	Src *dagger.Directory
}

func New(
	ctx context.Context,

	// The source directory
	src *dagger.Directory,

	// The sdk version
	// +default="latest"
	sdkVersion string,

	// The controller gen version
	// +default="latest"
	controllerGenVersion string,

	// The kustomize version to use
	// +default="latest"
	kustomizeVersion string,

	// The clean CRD version to use
	// +default="latest"
	cleanCrdVersion string,

	// The container to use with operator-sdk
	// +optional
	sdkContainer *dagger.Container,
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
