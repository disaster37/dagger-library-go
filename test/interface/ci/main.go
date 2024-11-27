// A generated module for Ci functions
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
	"dagger/ci/internal/dagger"
)

type Ci struct {
}

type Interface interface {
	DaggerObject
	ContainerEcho(ctx context.Context, stringArg string) *dagger.Container
}

// Returns a container that echoes whatever string argument is provided
func (m *Ci) ContainerEcho(ctx context.Context, interfacer Interface) *dagger.Container {
	return interfacer.ContainerEcho(ctx, "plop")

}
