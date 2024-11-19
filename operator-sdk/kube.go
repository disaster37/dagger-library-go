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

func (h *Kube) Cluster(
	ctx context.Context,
) (*dagger.Service, error) {
	service, err := h.K3s.
		Server().
		Start(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when start K3s")
	}

	// Install CRD on kube
	crdFile := h.Container.WithExec(helper.ForgeCommand("kustomize build config/crd -o /tmp/crd.yaml")).File("/tmp/crd.yaml")
	_, err = h.K3s.Kubectl("version").
		WithFile("/tmp/crd.yaml", crdFile).
		WithExec(helper.ForgeCommand("kubectl apply --server-side=true -f /tmp/crd.yaml")).
		Stdout(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when install CRDs")
	}

	return service, nil
}

func (h *Kube) Run(
	ctx context.Context,
) (*dagger.Service, error) {

	// Start k3s
	_, err := h.Cluster(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when start K3s")
	}

	// Run operator as service
	return h.Container.
		WithFile("/tmp/kubeconfig", h.K3s.Config()).
		WithEnvVariable("KUBECONFIG", "/tmp/kubeconfig").
		WithExposedPort(8081, dagger.ContainerWithExposedPortOpts{Protocol: dagger.NetworkProtocolTcp, Description: "Health"}).
		WithEnvVariable("ENABLE_WEBHOOKS", "false").
		WithEnvVariable("LOG_LEVEL", "trace").
		WithEnvVariable("LOG_FORMATTER", "json").
		WithExec(helper.ForgeCommand("go run cmd/main.go")).
		AsService().
		Start(ctx)
}

func (h *Kube) Kubeconfig() *dagger.File {
	return h.K3s.Config(dagger.K3SConfigOpts{Local: true})
}

func (h *Kube) Kubectl() *dagger.Container {
	return h.K3s.Kubectl("get nodes").
		WithDirectory("/project", h.Container.Directory(".")).
		WithWorkdir("/project")
}
