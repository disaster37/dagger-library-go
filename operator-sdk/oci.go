package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/helper"
)

type Auth struct {
	Url      string
	Username string
	Password *dagger.Secret
}

type Oci struct {
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
	Auths []Auth
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
) *Oci {
	return &Oci{
		Src:             src,
		GolangContainer: golangContainer.WithDirectory(".", src),
		DockerContainer: dockerContainer.WithDirectory(".", src),
		Auths:           make([]Auth, 0),
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
) *Oci {

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
func (h *Oci) PublishManager(
	ctx context.Context,

	// The image name to push
	// +required
	name string,
) (string, error) {

	if h.Manager == nil {
		h.BuildManager(ctx)
	}

	for _, auth := range h.Auths {
		h.Manager = h.Manager.WithRegistryAuth(auth.Url, auth.Username, auth.Password)
	}

	return h.Manager.Publish(ctx, name)

}

// BuildCatalog permit to build catalog image
func (h *Oci) BuildBundle(
	ctx context.Context,
) *Oci {
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
func (h *Oci) PublishBundle(
	ctx context.Context,

	// The image name to push
	// +required
	name string,
) (string, error) {

	if h.Bundle == nil {
		h.BuildBundle(ctx)
	}

	for _, auth := range h.Auths {
		h.Bundle = h.Bundle.WithRegistryAuth(auth.Url, auth.Username, auth.Password)
	}

	return h.Bundle.Publish(ctx, name)

}

func (h *Oci) BuildCatalog(
	ctx context.Context,

	// The catalog image name
	// +required
	catalogImage string,

	// The previuous catalog image name
	// If update 'true' and 'previousCatalogImage' not provided, it use the 'catalogImage'
	// +optional
	previousCatalogImage string,

	// The bundle image name
	// +required
	bundleImage string,

	// Set to true to update existing catalog
	// +optional
	update bool,
) (*Oci, error) {

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
	if update {
		if previousCatalogImage == "" {
			opmCmd = append(opmCmd,
				"--from-index",
				catalogImage,
			)
		} else {
			opmCmd = append(opmCmd,
				"--from-index",
				previousCatalogImage,
			)
		}
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
func (h *Oci) PublishCatalog(
	ctx context.Context,

	// The image name to push
	// +required
	name string,
) (string, error) {

	for _, auth := range h.Auths {
		h.Catalog = h.Catalog.WithRegistryAuth(auth.Url, auth.Username, auth.Password)
	}

	return h.Catalog.Publish(ctx, name)
}

// WithSource permit to update the current source
func (h *Oci) WithSource(
	// The source directory
	// +required
	src *dagger.Directory,
) *Oci {
	h.Src = src
	h.GolangContainer = h.GolangContainer.WithDirectory(".", src)
	h.DockerContainer = h.DockerContainer.WithDirectory(".", src)
	return h
}
