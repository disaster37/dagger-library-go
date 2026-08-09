// Runs a simulated Kubernetes cluster via KWOK (Kubernetes WithOut Kubelet)
// for fast operator and manifest testing. No real containers run; the
// kwok-controller drives pod status transitions from Stage rules, giving
// operators a full API surface to reconcile against with near-zero footprint.

package main

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"dagger/kwok/internal/dagger"

	"github.com/disaster37/dagger-library-go/lib/helper"
)

// entrypoint to setup cgroup nesting since kwok only does it
// when running as PID 1. This doesn't happen in Dagger given that we're using
// our custom shim
const entrypoint = `#!/bin/sh

set -o errexit
set -o nounset

#########################################################################################################################################
# DISCLAIMER																																																														#
# Copied from https://github.com/moby/moby/blob/ed89041433a031cafc0a0f19cfe573c31688d377/hack/dind#L28-L37															#
# Permission granted by Akihiro Suda <akihiro.suda.cz@hco.ntt.co.jp> (https://github.com/k3d-io/k3d/issues/493#issuecomment-827405962)	#
# Moby License Apache 2.0: https://github.com/moby/moby/blob/ed89041433a031cafc0a0f19cfe573c31688d377/LICENSE														#
#########################################################################################################################################
if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
  echo "[$(date -Iseconds)] [CgroupV2 Fix] Evacuating Root Cgroup ..."
  # move the processes from the root group to the /init group,
  # otherwise writing subtree_control fails with EBUSY.
  mkdir -p /sys/fs/cgroup/init
  xargs -rn1 < /sys/fs/cgroup/cgroup.procs > /sys/fs/cgroup/init/cgroup.procs || :
  # enable controllers
  sed -e 's/ / +/g' -e 's/^/+/' <"/sys/fs/cgroup/cgroup.controllers" >"/sys/fs/cgroup/cgroup.subtree_control"
  echo "[$(date -Iseconds)] [CgroupV2 Fix] Done"
fi

exec "$@"
`

// podLifecycleStages is the set of KWOK Stage resources that drive the
// simulated pod lifecycle: Pending -> ContainerCreating -> Running/Ready,
// plus init containers, completion (for Jobs), and deletion via finalizers.
// The ~1-6s delay per transition keeps intermediate states observable for
// operators (e.g. during a rolling update).
const podLifecycleStages = `
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
`

const (
	// defaultImage is the all-in-one kwok cluster image with kwokctl and all
	// component binaries pre-downloaded into /root/.kwok/cache.
	defaultImage = "registry.k8s.io/kwok/cluster:v0.6.1-k8s.v1.30.4"

	// kubeconfigPathFmt is the path to the in-cluster kubeconfig written by
	// kwokctl for cluster <name>.
	kubeconfigPathFmt = "/root/.kwok/clusters/%s/kubeconfig.yaml"

	// pollInterval is the delay between readiness polls.
	pollInterval = 2 * time.Second

	// apiserverReadyTimeout is the deadline (in seconds) for the API server to
	// answer after the cluster service is started.
	apiserverReadyTimeout = 120

	// maxNameLength is the DNS-1123 label limit for cluster/deployment
	// names and namespaces.
	maxNameLength = 63

	// maxNodes bounds the number of simulated nodes a cluster may scale to,
	// so a malformed request cannot exhaust the cluster container (CWE-400).
	maxNodes = 1000

	// maxReplicas bounds the number of simulated pods created by
	// SimulateRollingUpdate, so a malformed request cannot exhaust the
	// cluster container's etcd (CWE-400).
	maxReplicas = 1000

	// maxImageRefLength bounds image reference length (registry path +
	// tag/digest) to a sane size.
	maxImageRefLength = 512
)

var (
	// dns1123LabelRe matches Kubernetes DNS-1123 labels. It is the allowlist
	// used to validate every name before it is interpolated into shell
	// scripts, kubectl argument lists, YAML manifests or file paths
	// (CWE-20 / CWE-78 / CWE-74 / CWE-22).
	dns1123LabelRe = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

	// imageRefRe matches the container image reference grammar
	// (registry/path:tag@digest). It intentionally excludes whitespace,
	// quotes and every shell/YAML metacharacter so a validated reference can
	// neither inject shell commands nor additional YAML structure (CWE-20).
	imageRefRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/@-]*$`)
)

type Kwok struct {
	// +private
	Name string

	// +private
	ClusterCache *dagger.CacheVolume

	// +private
	Nodes int

	Container *dagger.Container
}

// New prepares a kwok cluster container. The cluster itself is created and
// started lazily by Server(); New only wires the image, the cgroupV2
// entrypoint, the per-name cluster cache volume, the pod-lifecycle Stage
// config, and the exposed API port.
func New(
	// The cluster name
	name string,

	// The number of nodes (1 default node when <= 1)
	// +optional
	nodes int,

	// The kwok cluster image to use
	// +optional
	// +default="registry.k8s.io/kwok/cluster:v0.6.1-k8s.v1.30.4"
	image string,
) (*Kwok, error) {
	// Security (CWE-20, CWE-78, CWE-22): the cluster name is interpolated
	// into the service startup shell script, used as the cache-volume key,
	// and embedded in the filesystem mount target
	// (/root/.kwok/clusters/<name>). Restrict it to a DNS-1123 label so it
	// can never carry shell metacharacters (command injection) or path
	// separators / ".." (path traversal). The `image` parameter needs no
	// such check here: it is only passed to dag.Container().From(), which
	// validates image reference syntax itself and never reaches a shell.
	if err := validateK8sName("cluster name", name); err != nil {
		return nil, err
	}
	// Security (CWE-400): bound the simulated node count to prevent resource
	// exhaustion of the cluster container (unbounded Node objects in etcd).
	if nodes > maxNodes {
		return nil, fmt.Errorf("nodes must be <= %d, got %d", maxNodes, nodes)
	}
	clusterCache := dag.CacheVolume("kwok_cluster_" + name)
	ctr := dag.Container().
		From(image).
		WithNewFile("/usr/bin/entrypoint.sh", entrypoint, dagger.ContainerWithNewFileOpts{
			Permissions: 0o755,
		}).
		WithEntrypoint([]string{"entrypoint.sh"}).
		WithMountedCache("/root/.kwok/clusters/"+name, clusterCache).
		WithNewFile("/kwok/config.yaml", podLifecycleStages).
		WithExposedPort(6443)
	return &Kwok{
		Name:         name,
		ClusterCache: clusterCache,
		Nodes:        nodes,
		Container:    ctr,
	}, nil
}

// Returns the kwok cluster as a service. Starting the service creates (on
// first start) or restarts (from persisted state) the cluster, waits for the
// API server, scales the cluster to the requested node count (kwok creates
// no nodes by default), rewrites the kubeconfig to the container IP, and
// tails the kwok-controller logs to keep the service alive. If the cluster
// is already being served by another service instance, starting fails with
// an explicit error (only one instance may serve a given cluster name).
func (m *Kwok) Server(
	// Alternative cluster CIDR (k3s parity; wired via kube-controller-manager extra args)
	// +optional
	clusterCidr string,

	// Alternative service CIDR (k3s parity; wired via apiserver/controller-manager extra args)
	// +optional
	serviceCird string,
) *dagger.Service {
	// Security (CWE-20, CWE-78): the CIDR values are interpolated into the
	// startup shell script (kwokctl --extra-args). Validate them as CIDRs
	// before use; net.ParseCIDR's grammar contains no shell metacharacters.
	// Builder-style functions cannot return errors, so invalid input yields
	// a service that fails fast at startup with a clear message (fail
	// closed, never fail open).
	if clusterCidr != "" {
		if err := validateCIDR("clusterCidr", clusterCidr); err != nil {
			return m.invalidOptService(err)
		}
	}
	if serviceCird != "" {
		if err := validateCIDR("serviceCird", serviceCird); err != nil {
			return m.invalidOptService(err)
		}
	}
	script := m.forgeServerScript(clusterCidr, serviceCird)
	return m.Container.AsService(dagger.ContainerAsServiceOpts{
		UseEntrypoint:            true,
		Args:                     []string{"sh", "-c", script},
		InsecureRootCapabilities: true,
	})
}

// invalidOptService returns a service that fails immediately at startup,
// surfacing the given validation error. It lets builder-style functions
// (which cannot return errors) fail closed on invalid input (CWE-754).
func (m *Kwok) invalidOptService(err error) *dagger.Service {
	// The message embeds the user-supplied value; shell-quote it so the
	// failure script itself cannot be injected (CWE-78).
	script := "echo " + shellQuote("ERROR: "+err.Error()) + " >&2; exit 1"
	return m.Container.AsService(dagger.ContainerAsServiceOpts{
		UseEntrypoint:            true,
		Args:                     []string{"sh", "-c", script},
		InsecureRootCapabilities: true,
	})
}

// Overwrite the current container
func (m *Kwok) WithContainer(c *dagger.Container) *Kwok {
	m.Container = c
	return m
}

// Run an arbitrary command inside the cluster container (builder-style, chainable)
func (m *Kwok) WithExec(command []string) *Kwok {
	m.Container = m.Container.WithExec(command)
	return m
}

// Returns the kubeconfig for the kwok cluster
func (m *Kwok) Config(ctx context.Context,
	// Rewrite the server URL to localhost (use with a published port, e.g. `dagger call server up`)
	// +optional
	// +default=false
	local bool,
) *dagger.File {
	return dag.Container().
		From("alpine").
		// bust cache so we always fetch the freshest kubeconfig
		WithEnvVariable("CACHE", time.Now().String()).
		WithMountedCache("/cache/kwok", m.ClusterCache).
		WithExec(helper.ForgeScript(
			"test -f /cache/kwok/kubeconfig.yaml && test -f /cache/kwok/pki/ca.crt && test -f /cache/kwok/pki/admin.crt && test -f /cache/kwok/pki/admin.key || { echo 'ERROR: kubeconfig not found - start the cluster first with `server`' >&2; exit 1; }\n" +
				"cp /cache/kwok/kubeconfig.yaml kubeconfig.yaml\n" +
				// kwokctl references the cluster PKI by file path, which only
				// resolves inside the cluster container; embed the certificates
				// so the kubeconfig is portable (engine-side containers, host
				// kubectl, operators under test).
				"CA=$(base64 /cache/kwok/pki/ca.crt | tr -d '\\n')\n" +
				"CERT=$(base64 /cache/kwok/pki/admin.crt | tr -d '\\n')\n" +
				"KEY=$(base64 /cache/kwok/pki/admin.key | tr -d '\\n')\n" +
				"sed -i \"s#certificate-authority: .*#certificate-authority-data: ${CA}#\" kubeconfig.yaml &&\n" +
				"sed -i \"s#client-certificate: .*#client-certificate-data: ${CERT}#\" kubeconfig.yaml &&\n" +
				"sed -i \"s#client-key: .*#client-key-data: ${KEY}#\" kubeconfig.yaml",
		)).
		With(func(c *dagger.Container) *dagger.Container {
			if local {
				c = c.WithExec([]string{"sed", "-i", `s#https://[0-9.]*:6443#https://localhost:6443#`, "kubeconfig.yaml"})
			}
			return c
		}).
		File("kubeconfig.yaml")
}

// Runs kubectl on the target kwok cluster.
//
// Security note (CWE-78, k3s parity): args is intentionally interpreted by a
// shell so callers can use the usual kubectl idioms (quoting, pipes). The
// command runs with the cluster-admin kubeconfig, so callers must never pass
// untrusted input as args. All other internal kubectl invocations in this
// module use exec form (no shell) instead.
func (m *Kwok) Kubectl(ctx context.Context, args string) *dagger.Container {
	return m.kubectlBase(ctx).WithExec([]string{"sh", "-c", "kubectl " + args})
}

// runs k9s on the target kwok cluster
func (m *Kwok) Kns(ctx context.Context) *dagger.Container {
	return dag.Container().
		From("derailed/k9s").
		WithoutEntrypoint().
		WithMountedCache("/cache/kwok", m.ClusterCache).
		WithEnvVariable("CACHE", time.Now().String()).
		WithEnvVariable("KUBECONFIG", "/.kube/config").
		WithFile("/.kube/config", m.Config(ctx, false), dagger.ContainerWithFileOpts{Owner: "1001"}).
		WithUser("1001").
		WithDefaultTerminalCmd([]string{"k9s"})
}

// kubectlBase is the shared kubectl container with the cluster kubeconfig
// mounted. It is reused by Kubectl, Apply, and waitForApiserver.
func (m *Kwok) kubectlBase(ctx context.Context) *dagger.Container {
	return dag.Container().
		From("bitnami/kubectl").
		WithoutEntrypoint().
		WithMountedCache("/cache/kwok", m.ClusterCache).
		WithEnvVariable("CACHE", time.Now().String()).
		WithFile("/.kube/config", m.Config(ctx, false), dagger.ContainerWithFileOpts{Owner: "1001"}).
		WithUser("1001")
}

// Apply a Kubernetes manifest to the cluster (provide exactly one of manifest or file)
func (m *Kwok) Apply(ctx context.Context,
	// Inline manifest YAML
	// +optional
	manifest string,

	// Manifest file
	// +optional
	file *dagger.File,
) (*dagger.Container, error) {
	hasManifest := manifest != ""
	hasFile := file != nil
	if hasManifest == hasFile {
		return nil, fmt.Errorf("provide exactly one of manifest or file")
	}
	if err := m.startAndWait(ctx); err != nil {
		return nil, err
	}
	ctr := m.kubectlBase(ctx)
	if hasManifest {
		ctr = ctr.WithNewFile("/tmp/manifest.yaml", manifest)
	} else {
		ctr = ctr.WithFile("/tmp/manifest.yaml", file, dagger.ContainerWithFileOpts{Owner: "1001"})
	}
	// Security (CWE-78): exec form (no shell) — the manifest content is only
	// ever read from a file and never interpolated into a command line. Note
	// that the manifest content itself is applied verbatim: that is the
	// purpose of this function, and it is safe because the target is a
	// disposable simulated test cluster.
	return ctr.WithExec([]string{"kubectl", "apply", "-f", "/tmp/manifest.yaml"}), nil
}

// Polls until the deployment reports all replicas ready
func (m *Kwok) WaitForDeploymentReady(ctx context.Context,
	// Deployment name
	name string,

	// Timeout in seconds
	// +optional
	// +default=300
	timeout int,

	// Namespace
	// +optional
	// +default="default"
	namespace string,
) error {
	// Security (CWE-20): validate the identifiers up front. Even though the
	// kubectl call below uses exec form (no shell), this keeps invalid names
	// away from kubectl argument parsing and any downstream YAML rendering.
	if err := validateK8sName("deployment name", name); err != nil {
		return err
	}
	if err := validateK8sName("namespace", namespace); err != nil {
		return err
	}
	if timeout <= 0 {
		return fmt.Errorf("timeout must be > 0, got %d", timeout)
	}
	if err := m.startServer(ctx); err != nil {
		return err
	}
	// Security (CWE-78): exec form, no shell — name and namespace are passed
	// as discrete arguments, so they can never inject shell commands. The
	// DNS-1123 validation above also rules out a leading "-", preventing
	// kubectl option injection (CWE-88).
	statusCmd := []string{
		"kubectl", "get", "deployment", name,
		"-n", namespace,
		"--request-timeout=10s",
		"-o", `jsonpath={.spec.replicas}|{.status.readyReplicas}|{.status.updatedReplicas}|{.status.observedGeneration}|{.metadata.generation}`,
	}
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	var lastErr error
	for !time.Now().After(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		stdout, err := m.kubectlBase(ctx).WithExec(statusCmd).Stdout(ctx)
		if err != nil {
			lastErr = err
		} else if ready, perr := isDeploymentReady(stdout); perr != nil {
			lastErr = perr
		} else if ready {
			return nil
		}
		// ctx-aware sleep so cancellation is honoured promptly (no unbounded
		// wait: the deadline and ctx together bound this loop, CWE-400).
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("not ready")
	}
	// %v (not %w): the Dagger SDK replaces returned errors that wrap an
	// ExecError with the bare ExecError, which would drop this context.
	return fmt.Errorf("deployment %q in namespace %q not ready after %ds: %v", name, namespace, timeout, lastErr)
}

// Creates a Deployment, waits for it, updates its image and waits for the
// simulated rolling update to complete
func (m *Kwok) SimulateRollingUpdate(ctx context.Context,
	// Deployment name
	name string,

	// Initial container image
	image string,

	// Number of replicas
	replicas int,

	// Image to roll out
	newImage string,

	// Timeout in seconds for each readiness wait
	// +optional
	// +default=300
	timeout int,

	// Namespace
	// +optional
	// +default="default"
	namespace string,
) (*dagger.Container, error) {
	// Security (CWE-20): every user-supplied value is validated before it is
	// interpolated into the YAML manifest or passed to kubectl.
	if err := validateK8sName("deployment name", name); err != nil {
		return nil, err
	}
	// Security (CWE-20, CWE-74): the image references are embedded in the
	// generated YAML manifest; the strict image-reference allowlist excludes
	// newlines/quotes/shell metacharacters, preventing YAML structure
	// injection and any downstream command injection.
	if err := validateImageRef("image", image); err != nil {
		return nil, err
	}
	if err := validateImageRef("newImage", newImage); err != nil {
		return nil, err
	}
	if replicas <= 0 {
		return nil, fmt.Errorf("replicas must be > 0, got %d", replicas)
	}
	// Security (CWE-400): bound the replica count so a malformed request
	// cannot exhaust the cluster container (unbounded Pod objects in etcd).
	if replicas > maxReplicas {
		return nil, fmt.Errorf("replicas must be <= %d, got %d", maxReplicas, replicas)
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be > 0, got %d", timeout)
	}
	if err := validateK8sName("namespace", namespace); err != nil {
		return nil, err
	}
	if err := m.startAndWait(ctx); err != nil {
		return nil, err
	}
	manifest := forgeDeploymentManifest(name, image, replicas, namespace)
	// Security (CWE-78): all kubectl invocations below use exec form (no
	// shell), so user-supplied values can never inject shell commands.
	if _, err := m.kubectlBase(ctx).
		WithNewFile("/tmp/manifest.yaml", manifest).
		WithExec([]string{"kubectl", "apply", "-f", "/tmp/manifest.yaml"}).
		Sync(ctx); err != nil {
		return nil, fmt.Errorf("failed to apply deployment %q: %v", name, err)
	}
	if err := m.WaitForDeploymentReady(ctx, name, timeout, namespace); err != nil {
		return nil, fmt.Errorf("initial rollout of deployment %q failed: %v", name, err)
	}
	if _, err := m.kubectlBase(ctx).
		WithExec([]string{"kubectl", "set", "image", "deployment/" + name, name + "=" + newImage, "-n", namespace}).
		Sync(ctx); err != nil {
		return nil, fmt.Errorf("failed to trigger rolling update on deployment %q: %v", name, err)
	}
	if err := m.WaitForDeploymentReady(ctx, name, timeout, namespace); err != nil {
		return nil, fmt.Errorf("rolling update of deployment %q failed: %v", name, err)
	}
	return m.kubectlBase(ctx).WithExec([]string{"kubectl", "rollout", "status", "deployment/" + name, "-n", namespace}), nil
}

// forgeDeploymentManifest renders a Deployment with rolling-update strategy.
// Container name equals the deployment name so `kubectl set image` works.
//
// Security precondition (CWE-74): name, image and namespace are interpolated
// verbatim into YAML. It must only ever receive validated input: callers
// validate them with validateK8sName / validateImageRef, whose allowlists
// exclude every YAML/shell metacharacter, so no extra YAML structure can be
// injected through these fields.
func forgeDeploymentManifest(name, image string, replicas int, namespace string) string {
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %[1]s
  namespace: %[2]s
  labels:
    app: %[1]s
spec:
  replicas: %[3]d
  selector:
    matchLabels:
      app: %[1]s
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  template:
    metadata:
      labels:
        app: %[1]s
    spec:
      containers:
        - name: %[1]s
          image: %[4]s
`, name, namespace, replicas, image)
}

// forgeServerScript builds the Server() startup script.
//
// Security precondition (CWE-78): this script interpolates m.Name and the
// CIDR values verbatim into shell text. It must only ever receive validated
// input: m.Name is validated by New() (DNS-1123 label) and the CIDRs are
// validated by Server() (net.ParseCIDR) before this helper is invoked.
func (m *Kwok) forgeServerScript(clusterCidr, serviceCidr string) string {
	// kwokctl v0.6.1 prepends "--" itself when forwarding extra-args, so the
	// flag keys must be passed without a leading dash.
	var cidrArgs []string
	if clusterCidr != "" {
		cidrArgs = append(cidrArgs,
			"--extra-args", "kube-controller-manager=cluster-cidr="+clusterCidr,
			"--extra-args", "kube-controller-manager=allocate-node-cidrs=true",
		)
	}
	if serviceCidr != "" {
		cidrArgs = append(cidrArgs,
			"--extra-args", "kube-apiserver=service-cluster-ip-range="+serviceCidr,
			"--extra-args", "kube-controller-manager=service-cluster-ip-range="+serviceCidr,
		)
	}
	cidrExtra := strings.Join(cidrArgs, " ")
	if cidrExtra != "" {
		cidrExtra = " " + cidrExtra
	}

	// kwok creates no nodes by default; always scale to at least one node,
	// otherwise pods could never be scheduled.
	nodes := m.Nodes
	if nodes < 1 {
		nodes = 1
	}

	return fmt.Sprintf(`set -o errexit
set -o nounset

if ! command -v kwokctl >/dev/null 2>&1; then
  echo "ERROR: kwokctl not found in the container image. Use a kwok cluster image such as %[1]s" >&2
  exit 1
fi

# If the API server behind the persisted kubeconfig is already answering,
# another service instance is serving this cluster. Starting a second one
# would fight over the shared cluster state, so fail with a clear message.
if [ -f %[5]s ] && kwokctl --name %[2]s kubectl get --raw=/readyz >/dev/null 2>&1; then
  echo "ERROR: kwok cluster %[2]s is already being served by another service instance; stop that instance first" >&2
  exit 1
fi

# On restarts the persisted kubeconfig still points at the previous
# container's IP; point it back at localhost so kwokctl's own checks (and
# ours) work inside this container while the cluster comes up.
if [ -f %[5]s ]; then
  sed -i "s#https://[0-9.]*:6443#https://127.0.0.1:6443#" %[5]s
fi

# Create the cluster on first start; restart it from persisted state afterwards.
# --wait blocks until every cluster component reports ready.
if ! kwokctl create cluster --runtime binary --name %[2]s \
      --kube-apiserver-port 6443 --kube-apiserver-insecure-port 0 \
      --wait %[6]ds \
      -c /kwok/config.yaml%[3]s; then
  echo "kwokctl create failed; trying to start existing cluster" >&2
  kwokctl start cluster --name %[2]s
fi

# Fail fast if the API server never answers.
i=0
until kwokctl --name %[2]s kubectl get --raw=/readyz >/dev/null 2>&1; do
  i=$((i+1))
  if [ "$i" -ge %[6]d ]; then
    echo "ERROR: API server of cluster %[2]s never became ready" >&2
    exit 1
  fi
  sleep 1
done

# Scale the cluster to the requested node count.
kwokctl scale node node --name %[2]s --replicas %[4]d

# Publish the engine-network IP in the kubeconfig so other containers can reach the cluster.
IP=$(ip route | grep src | awk '{print $NF}')
if [ -z "$IP" ]; then
  echo "ERROR: unable to determine container IP for kubeconfig rewrite" >&2
  exit 1
fi
sed -i "s#https://[0-9.]*:6443#https://${IP}:6443#" %[5]s

# Keep the service alive and surface controller logs.
exec kwokctl logs kwok-controller -f --name %[2]s
`,
		defaultImage, m.Name, cidrExtra, nodes, fmt.Sprintf(kubeconfigPathFmt, m.Name), apiserverReadyTimeout,
	)
}

// isDeploymentReady parses the jsonpath output
// "<spec>|<ready>|<updated>|<observedGeneration>|<generation>".
// Empty numeric fields count as 0. Ready when spec>0 and
// ready==spec && updated==spec && observedGeneration==generation.
func isDeploymentReady(statusLine string) (bool, error) {
	parts := strings.Split(statusLine, "|")
	if len(parts) != 5 {
		return false, fmt.Errorf("malformed status line %q: expected 5 fields separated by '|', got %d", statusLine, len(parts))
	}
	fieldNames := []string{"spec.replicas", "readyReplicas", "updatedReplicas", "observedGeneration", "generation"}
	values := make([]int, len(parts))
	for i, part := range parts {
		value, err := parseIntField(part)
		if err != nil {
			return false, fmt.Errorf("invalid %s %q: %w", fieldNames[i], part, err)
		}
		values[i] = value
	}
	spec, ready, updated, observed, generation := values[0], values[1], values[2], values[3], values[4]
	if spec <= 0 {
		return false, nil
	}
	return ready == spec && updated == spec && observed == generation, nil
}

// parseIntField parses a jsonpath field as an int, treating the empty string
// (a missing status field) as 0.
func parseIntField(s string) (int, error) {
	if s = strings.TrimSpace(s); s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

// validateK8sName validates a Kubernetes identifier (cluster name, deployment
// name, namespace) as a DNS-1123 label.
//
// Security (CWE-20): these values are interpolated into shell scripts, kubectl
// argument lists, YAML manifests and file paths. Restricting them to
// lower-case alphanumerics and '-' strips every shell/YAML metacharacter
// (preventing command injection CWE-78 and YAML injection CWE-74), forbids a
// leading '-' (preventing option/flag injection CWE-88), and forbids '/', '.'
// sequences (preventing path traversal CWE-22 when the value becomes part of
// the cluster cache mount path).
func validateK8sName(kind, value string) error {
	if value == "" {
		return fmt.Errorf("%s must not be empty", kind)
	}
	if len(value) > maxNameLength {
		return fmt.Errorf("%s %q must be at most %d characters (got %d)", kind, value, maxNameLength, len(value))
	}
	if !dns1123LabelRe.MatchString(value) {
		return fmt.Errorf("%s %q is invalid: must consist of lower case alphanumeric characters or '-', and start and end with an alphanumeric character", kind, value)
	}
	return nil
}

// validateImageRef validates a container image reference.
//
// Security (CWE-20, CWE-74): the value is embedded in a generated YAML
// manifest and passed to kubectl. The allowed character set
// (alphanumerics plus '.', '_', '-', ':', '/', '@') covers
// registry/path:tag@digest references while excluding whitespace, quotes and
// every shell/YAML metacharacter, so the value can inject neither additional
// YAML structure nor shell commands.
func validateImageRef(kind, value string) error {
	if value == "" {
		return fmt.Errorf("%s must not be empty", kind)
	}
	if len(value) > maxImageRefLength {
		return fmt.Errorf("%s %q must be at most %d characters (got %d)", kind, value, maxImageRefLength, len(value))
	}
	if !imageRefRe.MatchString(value) {
		return fmt.Errorf("%s %q is not a valid image reference (only alphanumerics and '.', '_', '-', ':', '/', '@' are allowed)", kind, value)
	}
	return nil
}

// validateCIDR validates a CIDR notation network range.
//
// Security (CWE-20): the value is interpolated into the service startup shell
// script (kwokctl --extra-args). net.ParseCIDR restricts it to the
// address/prefix grammar, which contains no shell metacharacters.
func validateCIDR(kind, value string) error {
	if _, _, err := net.ParseCIDR(value); err != nil {
		return fmt.Errorf("%s %q is not a valid CIDR: %v", kind, value, err)
	}
	return nil
}

// shellQuote single-quotes a value for safe interpolation into a POSIX shell
// command (CWE-78). It is used only for error messages that embed user input.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// startServer starts the cluster service (idempotent). If the cluster is
// already being served (e.g. by `server up` running in another session), it
// does nothing: starting a second service instance would fight the first one
// over the shared cluster state.
func (m *Kwok) startServer(ctx context.Context) error {
	if m.isClusterServed(ctx) {
		return nil
	}
	if _, err := m.Server("", "").Start(ctx); err != nil {
		return fmt.Errorf("failed to start kwok cluster %q: %w", m.Name, err)
	}
	return nil
}

// isClusterServed reports whether the API server behind the persisted
// kubeconfig is answering. A missing kubeconfig, an unreachable server, or
// an unhealthy API server all report false.
func (m *Kwok) isClusterServed(ctx context.Context) bool {
	// Security (CWE-78): exec form, no shell — the command is a fixed
	// argument vector with no user input.
	_, err := m.kubectlBase(ctx).
		WithExec([]string{"kubectl", "get", "--raw=/readyz", "--request-timeout=5s"}).
		Sync(ctx)
	return err == nil
}

// startAndWait starts the cluster service and waits for the API server to
// answer.
func (m *Kwok) startAndWait(ctx context.Context) error {
	if err := m.startServer(ctx); err != nil {
		return err
	}
	return m.waitForApiserver(ctx, apiserverReadyTimeout)
}

// waitForApiserver polls `kubectl get nodes` until it succeeds or the
// deadline passes; tolerates the kubeconfig not existing yet (retries).
func (m *Kwok) waitForApiserver(ctx context.Context, timeoutSecs int) error {
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	var lastErr error
	for !time.Now().After(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// Security (CWE-78): exec form, no shell — the command is a fixed
		// argument vector with no user input.
		_, lastErr = m.kubectlBase(ctx).
			WithExec([]string{"kubectl", "get", "nodes", "--request-timeout=10s"}).
			Stdout(ctx)
		if lastErr == nil {
			return nil
		}
		// ctx-aware sleep so cancellation is honoured promptly.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("apiserver not ready")
	}
	// %v (not %w): the Dagger SDK replaces returned errors that wrap an
	// ExecError with the bare ExecError, which would drop this context.
	return fmt.Errorf("apiserver not ready after %ds: %v", timeoutSecs, lastErr)
}
