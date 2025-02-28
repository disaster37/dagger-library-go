// A generated module for Kwok functions
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
	"dagger/kwok/internal/dagger"
	"fmt"
	"time"

	"github.com/disaster37/dagger-library-go/lib/helper"
)

type Kwok struct {
	// +private
	Name string

	// +private
	ClusterCache *dagger.CacheVolume

	Container *dagger.Container
}

func New(
	ctx context.Context,
	// The cluster name
	name string,

	// The number of nodes
	// +optional
	nodes int,
	// The alternative image to use
	// +optional
	// +default="alpine/curl:latest"
	image string,
	// The kwok version to use
	// +optional
	kwokVersion string,
) *Kwok {
	clusterCache := dag.CacheVolume("kwok_cluster_" + name)
	binCache := dag.CacheVolume("kwok_bin")
	kubeCache := dag.CacheVolume("kwok_kube")

	ctr := dag.Container().
		From(image).
		WithMountedCache("/cache/bin", binCache).
		WithMountedCache("/root/.kwok/clusters/"+name, clusterCache).
		WithMountedCache("/root/.kwok/cache", kubeCache).
		WithEnvVariable("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/cache/bin").
		WithExposedPort(6443)

	// install kwok if not yet exist
	if _, err := dag.Container().From(image).WithExec([]string{"kwokctl", "--version"}).Sync(ctx); err != nil {
		var urlKwok string
		if kwokVersion == "latest" || kwokVersion == "" {
			urlKwok = "https://github.com/kubernetes-sigs/kwok/releases/latest/download/kwokctl-linux-amd64"
		} else {
			urlKwok = fmt.Sprintf("https://github.com/kubernetes-sigs/kwok/releases/download/%s/kwokctl-linux-amd64", kwokVersion)
		}

		ctr = ctr.
			WithExec(helper.ForgeCommandf("curl -L %s -o /cache/bin/kwokctl", urlKwok)).
			WithExec(helper.ForgeCommand("chmod +x /cache/bin/kwokctl"))
	}
	ctr = ctr.WithNewFile("/kwok/config.yaml", `
kind: Stage
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: pod-create
spec:
  resourceRef:
    apiGroup: v1
    kind: Pod
  selector:
    matchExpressions:
      - key: '.metadata.deletionTimestamp'
        operator: 'DoesNotExist'
      - key: '.status.podIP'
        operator: 'DoesNotExist'
  weight: 1
  delay:
    durationMilliseconds: 1000
    jitterDurationMilliseconds: 5000
  next:
    event:
      type: Normal
      reason: Created
      message: Created container
    finalizers:
      add:
        - value: 'kwok.x-k8s.io/fake'
    statusTemplate: |
      {{ $now := Now }}

      conditions:
      {{ if .spec.initContainers }}
      - lastProbeTime: null
        lastTransitionTime: {{ $now | Quote }}
        message: 'containers with incomplete status: [{{ range .spec.initContainers }} {{ .name }} {{ end }}]'
        reason: ContainersNotInitialized
        status: "False"
        type: Initialized
      {{ else }}
      - lastProbeTime: null
        lastTransitionTime: {{ $now | Quote }}
        status: "True"
        type: Initialized
      {{ end }}
      - lastProbeTime: null
        lastTransitionTime: {{ $now | Quote }}
        message: 'containers with unready status: [{{ range .spec.containers }} {{ .name }} {{ end }}]'
        reason: ContainersNotReady
        status: "False"
        type: Ready
      - lastProbeTime: null
        lastTransitionTime: {{ $now | Quote }}
        message: 'containers with unready status: [{{ range .spec.containers }} {{ .name }} {{ end }}]'
        reason: ContainersNotReady
        status: "False"
        type: ContainersReady
      {{ range .spec.readinessGates }}
      - lastTransitionTime: {{ $now | Quote }}
        status: "True"
        type: {{ .conditionType | Quote }}
      {{ end }}

      {{ if .spec.initContainers }}
      initContainerStatuses:
      {{ range .spec.initContainers }}
      - image: {{ .image | Quote }}
        name: {{ .name | Quote }}
        ready: false
        restartCount: 0
        started: false
        state:
          waiting:
            reason: PodInitializing
      {{ end }}
      containerStatuses:
      {{ range .spec.containers }}
      - image: {{ .image | Quote }}
        name: {{ .name | Quote }}
        ready: false
        restartCount: 0
        started: false
        state:
          waiting:
            reason: PodInitializing
      {{ end }}
      {{ else }}
      containerStatuses:
      {{ range .spec.containers }}
      - image: {{ .image | Quote }}
        name: {{ .name | Quote }}
        ready: false
        restartCount: 0
        started: false
        state:
          waiting:
            reason: ContainerCreating
      {{ end }}
      {{ end }}

      hostIP: {{ NodeIPWith .spec.nodeName | Quote }}
      podIP: {{ PodIPWith .spec.nodeName ( or .spec.hostNetwork false ) ( or .metadata.uid "" ) ( or .metadata.name "" ) ( or .metadata.namespace "" ) | Quote }}
      phase: Pending
---
kind: Stage
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: pod-init-container-running
spec:
  resourceRef:
    apiGroup: v1
    kind: Pod
  selector:
    matchExpressions:
      - key: '.metadata.deletionTimestamp'
        operator: 'DoesNotExist'
      - key: '.status.phase'
        operator: 'In'
        values:
          - 'Pending'
      - key: '.status.conditions.[] | select( .type == "Initialized" ) | .status'
        operator: 'NotIn'
        values:
          - 'True'
      - key: '.status.initContainerStatuses.[].state.waiting.reason'
        operator: 'Exists'
  weight: 1
  delay:
    durationMilliseconds: 1000
    jitterDurationMilliseconds: 5000
  next:
    statusTemplate: |
      {{ $now := Now }}
      {{ $root := . }}
      initContainerStatuses:
      {{ range $index, $item := .spec.initContainers }}
      {{ $origin := index $root.status.initContainerStatuses $index }}
      - image: {{ $item.image | Quote }}
        name: {{ $item.name | Quote }}
        ready: true
        restartCount: 0
        started: true
        state:
          running:
            startedAt: {{ $now | Quote }}
      {{ end }}
---
kind: Stage
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: pod-init-container-completed
spec:
  resourceRef:
    apiGroup: v1
    kind: Pod
  selector:
    matchExpressions:
      - key: '.metadata.deletionTimestamp'
        operator: 'DoesNotExist'
      - key: '.status.phase'
        operator: 'In'
        values:
          - 'Pending'
      - key: '.status.conditions.[] | select( .type == "Initialized" ) | .status'
        operator: 'NotIn'
        values:
          - 'True'
      - key: '.status.initContainerStatuses.[].state.running.startedAt'
        operator: 'Exists'
  weight: 1
  delay:
    durationMilliseconds: 1000
    jitterDurationMilliseconds: 5000
  next:
    statusTemplate: |
      {{ $now := Now }}
      {{ $root := . }}
      conditions:
      - lastProbeTime: null
        lastTransitionTime: {{ $now | Quote }}
        status: "True"
        type: Initialized
      initContainerStatuses:
      {{ range $index, $item := .spec.initContainers }}
      {{ $origin := index $root.status.initContainerStatuses $index }}
      - image: {{ $item.image | Quote }}
        name: {{ $item.name | Quote }}
        ready: true
        restartCount: 0
        started: false
        state:
          terminated:
            exitCode: 0
            finishedAt: {{ $now | Quote }}
            reason: Completed
            startedAt: {{ $now | Quote }}
      {{ end }}
      containerStatuses:
      {{ range .spec.containers }}
      - image: {{ .image | Quote }}
        name: {{ .name | Quote }}
        ready: false
        restartCount: 0
        started: false
        state:
          waiting:
            reason: ContainerCreating
      {{ end }}
---
kind: Stage
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: pod-ready
spec:
  resourceRef:
    apiGroup: v1
    kind: Pod
  selector:
    matchExpressions:
      - key: '.metadata.deletionTimestamp'
        operator: 'DoesNotExist'
      - key: '.status.phase'
        operator: 'In'
        values:
          - 'Pending'
      - key: '.status.conditions.[] | select( .type == "Initialized" ) | .status'
        operator: 'In'
        values:
          - 'True'
      - key: '.status.conditions.[] | select( .type == "ContainersReady" ) | .status'
        operator: 'NotIn'
        values:
          - 'True'
  weight: 1
  delay:
    durationMilliseconds: 1000
    jitterDurationMilliseconds: 5000
  next:
    delete: false
    statusTemplate: |
      {{ $now := Now }}
      {{ $root := . }}
      conditions:
      - lastProbeTime: null
        lastTransitionTime: {{ $now | Quote }}
        message: ''
        reason: ''
        status: "True"
        type: Ready
      - lastProbeTime: null
        lastTransitionTime: {{ $now | Quote }}
        message: ''
        reason: ''
        status: "True"
        type: ContainersReady
      containerStatuses:
      {{ range $index, $item := .spec.containers }}
      {{ $origin := index $root.status.containerStatuses $index }}
      - image: {{ $item.image | Quote }}
        name: {{ $item.name | Quote }}
        ready: true
        restartCount: 0
        started: true
        state:
          running:
            startedAt: {{ $now | Quote }}
      {{ end }}
      phase: Running
---
kind: Stage
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: pod-complete
spec:
  resourceRef:
    apiGroup: v1
    kind: Pod
  selector:
    matchExpressions:
      - key: '.metadata.deletionTimestamp'
        operator: 'DoesNotExist'
      - key: '.status.phase'
        operator: 'In'
        values:
          - 'Running'
      - key: '.status.conditions.[] | select( .type == "Ready" ) | .status'
        operator: 'In'
        values:
          - 'True'
      - key: '.metadata.ownerReferences.[].kind'
        operator: 'In'
        values:
          - 'Job'
  weight: 1
  delay:
    durationMilliseconds: 1000
    jitterDurationMilliseconds: 5000
  next:
    delete: false
    statusTemplate: |
      {{ $now := Now }}
      {{ $root := . }}
      containerStatuses:
      {{ range $index, $item := .spec.containers }}
      {{ $origin := index $root.status.containerStatuses $index }}
      - image: {{ $item.image | Quote }}
        name: {{ $item.name | Quote }}
        ready: true
        restartCount: 0
        started: false
        state:
          terminated:
            exitCode: 0
            finishedAt: {{ $now | Quote }}
            reason: Completed
            startedAt: {{ $now | Quote }}
      {{ end }}
      phase: Succeeded
---
kind: Stage
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: pod-remove-finalizer
spec:
  resourceRef:
    apiGroup: v1
    kind: Pod
  selector:
    matchExpressions:
      - key: '.metadata.deletionTimestamp'
        operator: 'Exists'
      - key: '.metadata.finalizers.[]'
        operator: 'In'
        values:
          - 'kwok.x-k8s.io/fake'
  weight: 1
  delay:
    durationMilliseconds: 1000
    jitterDurationMilliseconds: 5000
  next:
    finalizers:
      remove:
        - value: 'kwok.x-k8s.io/fake'
    event:
      type: Normal
      reason: Killing
      message: Stopping container
---
kind: Stage
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: pod-delete
spec:
  resourceRef:
    apiGroup: v1
    kind: Pod
  selector:
    matchExpressions:
      - key: '.metadata.deletionTimestamp'
        operator: 'Exists'
      - key: '.metadata.finalizers'
        operator: 'DoesNotExist'
  weight: 1
  delay:
    durationMilliseconds: 1000
    jitterDurationFrom:
      expressionFrom: '.metadata.deletionTimestamp'
  next:
    delete: true
`).
		WithExec(helper.ForgeCommandf("kwokctl create cluster --runtime binary --kube-apiserver-port 6443 -c /kwok/config.yaml --name %s", name)).
		Terminal()

	if nodes > 1 {
		ctr = ctr.WithExec(helper.ForgeCommandf("kwokctl scale node node --replicas %d", nodes))
	}

	return &Kwok{
		Name:         name,
		ClusterCache: clusterCache,
		Container:    ctr,
	}
}

// Returns a newly initialized Kwok cluster
func (m *Kwok) Server() *dagger.Service {
	return m.Container.
		WithEntrypoint(helper.ForgeCommandf("kwokctl logs kwok-controller -f --name %s", m.Name)).
		Terminal().
		AsService()
}

// Overwrite the current container
func (m *Kwok) WithContainer(c *dagger.Container) *Kwok {
	m.Container = c
	return m
}

// returns the config file for the k3s cluster
func (m *Kwok) Config(ctx context.Context,
	// +optional
	// +default=false
	local bool,
) *dagger.File {
	return dag.Container().
		From("alpine").
		// we need to bust the cache so we don't fetch the same file each time.
		WithEnvVariable("CACHE", time.Now().String()).
		WithMountedCache("/cache/kwok", m.ClusterCache).
		WithExec([]string{"cp", "/cache/kwok/kubeconfig.yaml", "kubeconfig.yaml"}).
		With(func(c *dagger.Container) *dagger.Container {
			if !local {
				// Get the current service endpoint
				endpoint, err := m.Container.WithExec(helper.ForgeScript("ip route | grep src | awk '{print $NF}'")).Stdout(ctx)
				if err != nil {
					panic(err)
				}
				c = c.WithExec([]string{"sed", "-i", fmt.Sprintf(`s/https:.*:6443/https:\/\/%s:6443/g`, endpoint), "kubeconfig.yaml"})
			}
			return c
		}).
		File("kubeconfig.yaml")
}

// runs kubectl on the target Kwok cluster
func (m *Kwok) Kubectl(ctx context.Context, args string) *dagger.Container {
	return dag.Container().
		From("bitnami/kubectl").
		WithoutEntrypoint().
		WithEnvVariable("CACHE", time.Now().String()).
		WithFile("/.kube/config", m.Config(ctx, false), dagger.ContainerWithFileOpts{Permissions: 1001}).
		WithUser("1001").
		WithExec([]string{"sh", "-c", "kubectl " + args})
}

// runs k9s on the target k3s cluster
func (m *Kwok) Kns(ctx context.Context) *dagger.Container {
	return dag.Container().
		From("derailed/k9s").
		WithoutEntrypoint().
		WithEnvVariable("CACHE", time.Now().String()).
		WithEnvVariable("KUBECONFIG", "/.kube/config").
		WithFile("/.kube/config", m.Config(ctx, false), dagger.ContainerWithFileOpts{Permissions: 1001}).
		Terminal().
		WithDefaultTerminalCmd([]string{"k9s"})
}
