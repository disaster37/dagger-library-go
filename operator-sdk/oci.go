package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/helper"
)

type OperatorSdkOciAuth struct {
	Url      string
	Username string
	Password *dagger.Secret
}

type OperatorSdkOci struct {
	// The Golang container
	GolangContainer *dagger.Container

	// The Docker container
	DockerContainer *dagger.Container

	// The siurce directory
	// +private
	Src *dagger.Directory

	// The manager image
	Manager *dagger.Container

	// The bundle image
	Bundle *dagger.Container

	// The catalog image
	Catalog *dagger.Container

	// +private
	Auths []OperatorSdkOciAuth
}

func NewOci(
	// The source directory
	// +required
	src *dagger.Directory,
	// The base container with golang
	// +required
	golangContainer *dagger.Container,
	// +required
	// The base container with docker
	dockerContainer *dagger.Container,
) *OperatorSdkOci {
	return &OperatorSdkOci{
		Src:             src,
		GolangContainer: golangContainer.WithDirectory(".", src),
		DockerContainer: dockerContainer.WithDirectory(".", src),
		Auths:           make([]OperatorSdkOciAuth, 0),
	}
}

func (h *OperatorSdkOci) WithRepositoryCredentials(
	// The repository URL
	// +required
	url string,

	// The username
	// +required
	username string,

	// The password
	// +required
	password *dagger.Secret,
) *OperatorSdkOci {
	h.Auths = append(h.Auths, OperatorSdkOciAuth{
		Url:      url,
		Username: username,
		Password: password,
	})

	return h
}

// BuildManager permit to build manager image
func (h *OperatorSdkOci) BuildManager(
	ctx context.Context,
) *OperatorSdkOci {

	// Build manager
	managerBinFile := h.GolangContainer.
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec(helper.ForgeCommand("go build -a -o manager cmd/main.go")).
		File("manager")

	h.Manager = dag.Container().
		From("gcr.io/distroless/static:nonroot").
		WithWorkdir("/").
		WithFile(".", managerBinFile).
		WithUser("65532:65532").
		WithEntrypoint([]string{"/manager"})

	return h

}

// PublishManager permit to push OCI image on registry
func (h *OperatorSdkOci) PublishManager(
	ctx context.Context,

	// The image name to push
	// +required
	name string,
) (string, error) {

	if h.Manager == nil {
		h.BuildManager(ctx)
	}

	manager := h.Manager

	for _, auth := range h.Auths {
		manager = manager.WithRegistryAuth(auth.Url, auth.Username, auth.Password)
	}

	return manager.Publish(ctx, name)

}

// BuildCatalog permit to build catalog image
func (h *OperatorSdkOci) BuildBundle(
	ctx context.Context,
) *OperatorSdkOci {
	h.Bundle = h.GolangContainer.
		Directory(".").
		DockerBuild(
			dagger.DirectoryDockerBuildOpts{
				Dockerfile: "bundle.Dockerfile",
			},
		)

	return h
}

// PublishBundle permit to push OCI image on registry
func (h *OperatorSdkOci) PublishBundle(
	ctx context.Context,

	// The image name to push
	// +required
	name string,
) (string, error) {

	if h.Bundle == nil {
		h.BuildBundle(ctx)
	}

	bundle := h.Bundle

	for _, auth := range h.Auths {
		bundle = bundle.WithRegistryAuth(auth.Url, auth.Username, auth.Password)
	}

	return bundle.Publish(ctx, name)

}

// Build the OLM catalog
func (h *OperatorSdkOci) BuildCatalog(
	ctx context.Context,

	// The catalog image name
	// +required
	catalogImage string,

	// The previuous catalog image name
	// +optional
	previousCatalogImage string,

	// The bundle image name
	// +required
	bundleImage string,
) (*OperatorSdkOci, error) {

	// Run OPM command
	opmCmd := []string{
		"opm",
		"index",
		"add",
		"--container-tool",
		"docker",
		"--mode",
		"semver",
		"--tag",
		catalogImage,
		"--bundles",
		bundleImage,
	}

	if previousCatalogImage != "" {
		opmCmd = append(opmCmd,
			"--from-index",
			previousCatalogImage,
		)
	}

	dockerContainer := h.DockerContainer.
		WithExec(opmCmd)
	_, err := dockerContainer.
		Stdout(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when create catalog image")
	}

	// Export docker image and import them on dagger
	catalogFile := dockerContainer.
		WithExec([]string{
			"docker",
			"save",
			"--output=/tmp/image.tar",
			catalogImage,
		}).File("/tmp/image.tar")

	h.Catalog = dag.Container().Import(catalogFile)

	return h, nil
}

// PublishCatalog permit to publish the catalog image
func (h *OperatorSdkOci) PublishCatalog(
	ctx context.Context,

	// The image name to push
	// +required
	name string,
) (string, error) {

	catalog := h.Catalog

	for _, auth := range h.Auths {
		catalog = catalog.WithRegistryAuth(auth.Url, auth.Username, auth.Password)
	}

	return catalog.Publish(ctx, name)
}

// WithSource permit to update the current source
func (h *OperatorSdkOci) WithSource(
	// The source directory
	// +required
	src *dagger.Directory,
) *OperatorSdkOci {
	h.Src = src
	h.GolangContainer = h.GolangContainer.WithDirectory(".", src)
	h.DockerContainer = h.DockerContainer.WithDirectory(".", src)
	return h
}
