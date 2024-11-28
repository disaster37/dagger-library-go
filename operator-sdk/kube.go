package main

import (
	"dagger/operator-sdk/internal/dagger"
)

type Kube struct {
	// +private
	Src *dagger.Directory

	// +private
	Kube *dagger.K3S
}

func NewKube(
	// The golang container
	src *dagger.Directory,
) *Kube {
	return &Kube{
		Src:  src,
		Kube: dag.K3S("test"),
	}
}

func (h *Kube) WithSource(
	// The source directory
	// +required
	src *dagger.Directory,
) *Kube {
	h.Src = src
	return h
}

func (h *Kube) Kubectl() *dagger.Container {
	return h.Kube.Kubectl("get nodes")
}

/*
func (h *Kube) Kubeconfig(
	// set true if expose the k3s on host
	// +optional
	local bool,
) *dagger.File {
	return h.K3S.Config(dagger.K3SConfigOpts{Local: local})
}



func (h *Kube) KubeCluster(
	ctx context.Context,
) (*dagger.Service, error) {
	return h.K3S.Server().Start(ctx)
}
*/

/*
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
*/
