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
	_, err := kServer.Start(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when start K3s")
	}

	// Install CRD on kube
	crdFile := h.Container.WithExec(helper.ForgeCommand("kustomize build config/crd -o /tmp/crd.yaml")).File("/tmp/crd.yaml")
	_, err = h.K3s.Kubectl("help").
		WithFile("/tmp/crd.yaml", crdFile).
		WithExec(helper.ForgeCommand("kubectl apply --server-side=true -f /tmp/crd.yaml")).
		Stdout(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when install CRDs")
	}

	// Run operator as service
	_, err = h.Container.
		WithFile("/tmp/kubeconfig", h.K3s.Config()).
		WithEnvVariable("KUBECONFIG", "/tmp/kubeconfig").
		WithExposedPort(8081, dagger.ContainerWithExposedPortOpts{Protocol: dagger.NetworkProtocolTcp, Description: "Health"}).
		WithEnvVariable("ENABLE_WEBHOOKS", "false").
		WithEnvVariable("LOG_LEVEL", "trace").
		WithEnvVariable("LOG_FORMATTER", "json").
		WithExec(helper.ForgeCommand("go run cmd/main.go")).
		AsService().
		Start(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when run operator")
	}

	return h.K3s.Kubectl("get nodes").
		WithDirectory("/project", h.Container.Directory(".")).
		WithWorkdir("/project"), nil

}
