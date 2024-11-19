package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/helper"
)

type Kube struct {
	Container *dagger.Container

	// +private
	K3s *dagger.K3S
}

func NewKube(
	// The golang container
	container *dagger.Container,
) *Kube {
	return &Kube{
		Container: container,
		K3s:       dag.K3S("test"),
	}
}

func (h *Kube) Run(
	ctx context.Context,
) (*dagger.Container, error) {

	// Start k3s
	kServer := h.K3s.Server()
	kServer, err := kServer.Start(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when start K3s")
	}
	defer func() {
		_, _ = kServer.Stop(ctx)
	}()

	// Install CRD on kube
	crdFile := h.Container.WithExec(helper.ForgeCommand("kustomize build config/crd -o /tmp/crd.yaml")).File("/tmp/crd.yaml")
	_, err = h.K3s.Kubectl("version").
		WithFile("/tmp/crd.yaml", crdFile).
		WithExec(helper.ForgeCommand("kubectl apply -f /tmp/crd.yaml")).
		Stdout(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when install CRDs")
	}

	// Run operator as service
	operatorService, err := h.Container.
		WithExposedPort(8081, dagger.ContainerWithExposedPortOpts{Protocol: dagger.NetworkProtocolTcp, Description: "Health"}).
		WithExec(helper.ForgeCommand("LOG_LEVEL=trace LOG_FORMATTER=json go run cmd/main.go")).
		AsService().
		Start(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when run operator")
	}
	defer func() {
		_, _ = operatorService.Stop(ctx)
	}()

	return h.K3s.Kns().
		WithDirectory("/project", h.Container.Directory(".")).
		WithWorkdir("/project"), nil

}
