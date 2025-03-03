package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"
)

type OperatorSdkKube struct {
	// +private
	Src *dagger.Directory

	// +private
	Kube *dagger.K3S
}

func NewKube(
	// The golang container
	src *dagger.Directory,
) *OperatorSdkKube {
	return &OperatorSdkKube{
		Src: src,
		Kube: dag.K3S("test").
			With(func(k *dagger.K3S) *dagger.K3S {
				return k.WithContainer(
					k.Container().
						WithExec([]string{"sh", "-c", `
cat <<EOF > /etc/rancher/k3s/registries.yaml
configs:
  "*":
    tls:
      insecure_skip_verify: true
EOF`}),
				)
			}),
	}
}

func (h *OperatorSdkKube) WithSource(
	// The source directory
	// +required
	src *dagger.Directory,
) *OperatorSdkKube {
	h.Src = src
	return h
}

func (h *OperatorSdkKube) Kubectl() *dagger.Container {
	return h.Kube.Kubectl("version").
		WithDirectory("src", h.Src).
		WithWorkdir("/src")
}

func (h *OperatorSdkKube) Kubeconfig(
	// set true if expose the k3s on host
	// +optional
	local bool,
) *dagger.File {
	return h.Kube.Config(dagger.K3SConfigOpts{Local: local})
}

func (h *OperatorSdkKube) KubeCluster(
	ctx context.Context,
) (*dagger.Service, error) {
	return h.Kube.Server(dagger.K3SServerOpts{
		ClusterCidr: "10.44.0.0/16",
		ServiceCird: "10.45.0.0/16",
	}).Start(ctx)
}

func (h *OperatorSdkKube) KubeContainer() *dagger.Container {
	return h.Kube.Container()
}

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
		WithEntrypoint(helper.ForgeCommand("go run cmd/main.go")).
		AsService().
		Start(ctx)
}
*/
