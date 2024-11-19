package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"

	"github.com/disaster37/dagger-library-go/lib/helper"
)

type Auth struct {
	Url      string
	Username string
	Password *dagger.Secret
}

type Oci struct {
	Container *dagger.Container

	// +private
	Manager *dagger.Container

	// +private
	Bundle *dagger.Container

	// +private
	Auths []Auth
}

func NewOci(
	// The base container with golang
	// +required
	container *dagger.Container,
) *Oci {
	return &Oci{
		Container: container,
		Auths:     make([]Auth, 0),
	}
}

func (h *Oci) WithRepositoryCredentials(
	// The repository URL
	// +required
	url string,

	// The username
	// +required
	username string,

	// The password
	// +required
	password *dagger.Secret,
) *Oci {
	h.Auths = append(h.Auths, Auth{
		Url:      url,
		Username: username,
		Password: password,
	})

	return h
}

// BuildManager permit to build manager image
func (h *Oci) BuildManager(
	ctx context.Context,
) *dagger.Container {

	// Build manager
	managerBinFile := h.Container.
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec(helper.ForgeCommand("go build -a -o manager cmd/main.go")).File("manager")

	h.Manager = dag.Container().
		From("gcr.io/distroless/static:nonroot").
		WithWorkdir("/").
		WithFile(".", managerBinFile).
		WithUser("65532:65532").
		WithEntrypoint([]string{"/manager"})

	return h.Manager

}

// PublishManager permit to push OCI image on registry
func (h *Oci) PublishManager(
	ctx context.Context,

	// The image name to push
	// +required
	name string,
) (string, error) {

	if h.Manager == nil {
		h.Manager = h.BuildManager(ctx)
	}

	for _, auth := range h.Auths {
		h.Manager = h.Manager.WithRegistryAuth(auth.Url, auth.Username, auth.Password)
	}

	return h.Manager.Publish(ctx, name)

}

// BuildCatalog permit to build catalog image
func (h *Oci) BuildBundle(
	ctx context.Context,
) *dagger.Container {
	h.Bundle = h.Container.Directory(".").DockerBuild(dagger.DirectoryDockerBuildOpts{
		Dockerfile: "bundle.Dockerfile",
	})

	return h.Bundle
}

// PublishBundle permit to push OCI image on registry
func (h *Oci) PublishBundle(
	ctx context.Context,

	// The image name to push
	// +required
	name string,
) (string, error) {

	if h.Bundle == nil {
		h.Bundle = h.BuildBundle(ctx)
	}

	for _, auth := range h.Auths {
		h.Bundle = h.Bundle.WithRegistryAuth(auth.Url, auth.Username, auth.Password)
	}

	return h.Manager.Publish(ctx, name)

}

// WithSource permit to update the current source
func (h *Oci) WithSource(
	// The source directory
	// +required
	src *dagger.Directory,
) *Oci {
	h.Container = h.Container.WithDirectory(".", src)
	return h
}
