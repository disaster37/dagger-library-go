# kwok module

A Dagger module that runs a **simulated Kubernetes cluster via [KWOK](https://github.com/kubernetes-sigs/kwok)** (Kubernetes WithOut Kubelet) for fast operator and manifest testing. No real containers run; the `kwok-controller` drives pod status transitions from `Stage` rules, giving operators a full API surface (a real `kube-apiserver`/`etcd`/`controller-manager`/`scheduler`) to reconcile against, with near-zero footprint.

## How it works

- **Cluster image**: uses the all-in-one `registry.k8s.io/kwok/cluster:<kwok-ver>-k8s.<k8s-ver>` image, which ships `kwok` + `kwokctl` and pre-downloads every component binary (etcd, kube-apiserver, kube-controller-manager, kube-scheduler, kubectl) into `/root/.kwok/cache`. Cluster creation therefore needs **no downloads**.
- **Stages-driven pod lifecycle**: a set of `Stage` resources (Pending → ContainerCreating → Running/Ready, plus init containers, completion for Jobs, and deletion via finalizers) is written to `/kwok/config.yaml` and passed to `kwokctl create cluster -c`. Each transition has a ~1–6s delay so intermediate states stay observable for operators (e.g. during a rolling update).
- **Service lifecycle**: the cluster is created and started **lazily** by `server` (mirrors the `k3s` module). Calling `server up` runs a startup script that creates (or restarts from persisted state) the cluster, waits for the API server, scales the cluster to the requested node count (kwok creates **no nodes by default**, so at least one node is always created), rewrites the kubeconfig to the container IP, and tails the `kwok-controller` logs to keep the service alive.
- **Kubeconfig persistence**: cluster state (etcd, PKI, kubeconfig) persists in a per-name Dagger cache volume mounted at `/root/.kwok/clusters/<name>`. Restarting the service reuses the persisted cluster; use a new `name` to reset.

## Requirements

- A Dagger CLI compatible with engine `v0.16.1`.

## Development

After code changes, regenerate the SDK bindings with `dagger develop`, then:

```bash
go build ./... && go vet ./...

# The generated SDK client init() expects a Dagger session; any dummy values
# work for the pure unit tests (no engine is contacted).
DAGGER_SESSION_PORT=1 DAGGER_SESSION_TOKEN=test go test ./...
```

## Quick start

Run from the `kwok/` module directory (regenerate SDK bindings after code changes with `dagger develop`):

```bash
# Start the cluster and publish port 6443 locally (blocks; use a second shell)
dagger call --name=itest server up

# Engine-side kubectl (from another shell)
dagger call --name=itest kubectl --args "get nodes"
dagger call --name=itest kubectl --args "get pods -A"

# Local kubeconfig (works against the published port from shell 1)
dagger call --name=itest config --local export --path ./kubeconfig.kwok
kubectl --kubeconfig ./kubeconfig.kwok get nodes

# Apply a manifest
dagger call --name=itest apply --manifest "$(cat <<'EOF'
apiVersion: v1
kind: ConfigMap
metadata:
  name: demo
data:
  key: value
EOF
)"

# Simulated rolling update (create nginx:1.25 x3, roll to nginx:1.26, wait)
dagger call --name=itest simulate-rolling-update \
  --name demo --image nginx:1.25 --replicas 3 --new-image nginx:1.26

# Standalone readiness wait
dagger call --name=itest wait-for-deployment-ready --name demo --timeout 120

# Interactive k9s
dagger call --name=itest kns
```

Expected: cluster creation logs show no binary downloads; nodes become Ready; pods transition Pending → ContainerCreating → Running during the rolling update; old pods terminate via the finalizer stages.

## Function reference

### `new` — Initialize

| Arg | Required | Default | Description |
|-----|----------|---------|-------------|
| `name` | yes | — | Cluster name (also the cache volume key). Must be a DNS-1123 label (lower-case alphanumerics and `-`, max 63 chars); rejected otherwise |
| `nodes` | no | `0` (1 node is created) | Number of nodes; kwok creates no nodes by default, so the startup script always scales to at least 1. Capped at `1000` |
| `image` | no | `registry.k8s.io/kwok/cluster:v0.6.1-k8s.v1.30.4` | kwok cluster image (must contain `kwokctl`) |

### `server` — Run the cluster as a service

| Arg | Required | Default | Description |
|-----|----------|---------|-------------|
| `clusterCidr` | no | `""` | Cluster CIDR (k3s parity; wired via `kube-controller-manager` extra args). Must be a valid CIDR; an invalid value makes the service fail fast at startup |
| `serviceCird` | no | `""` | Service CIDR (k3s parity; wired via apiserver/controller-manager extra args). Must be a valid CIDR; an invalid value makes the service fail fast at startup |

### `config` — Kubeconfig

| Arg | Required | Default | Description |
|-----|----------|---------|-------------|
| `local` | no | `false` | Rewrite the server URL to `localhost` (use with a published port, e.g. `server up`) |

### Other functions

| Function | Description |
|----------|-------------|
| `kubectl --args <args>` | Runs `kubectl <args>` against the cluster (engine-side, container-IP kubeconfig). `args` is interpreted by a shell (k3s parity) — never pass untrusted input |
| `kns` | Runs `k9s` against the cluster |
| `with-container` | Overwrite the internal container |
| `with-exec` | Run an arbitrary command in the cluster container (chainable) |
| `apply --manifest <yaml>` / `apply --file <file>` | Apply a Kubernetes manifest (provide exactly one) |
| `wait-for-deployment-ready --name <n> [--timeout <s>] [--namespace <ns>]` | Poll until a Deployment is fully ready. `name`/`namespace` must be DNS-1123 labels |
| `simulate-rolling-update --name --image --replicas --new-image [--timeout] [--namespace]` | Create a Deployment, wait, roll the image, wait again. Names/namespace must be DNS-1123 labels, images must be valid references, `replicas` is capped at `1000` |

## Operator-testing recipe

1. Start the cluster: `dagger call --name=itest server up` (or call `server` from another module and `.Start(ctx)`).
2. Apply CRDs/manifests: `dagger call --name=itest apply --file ./crds.yaml` (or `apply --manifest ...`).
3. Hand the kubeconfig to the operator under test: `m.Config(ctx, false)` for engine-side access, or `m.Config(ctx, true)` exported for host access through a published port.
4. Assert with `kubectl`: `dagger call --name=itest kubectl --args "get <resource>"`, or `m.WaitForDeploymentReady(ctx, ...)`.

## Notes & troubleshooting

- **Stage delays**: each pod transition takes ~1–6s. This is intentional so operators can observe intermediate states (Pending, ContainerCreating, RollingUpdate in progress).
- **Cluster state persists per `name`** in a Dagger cache volume (`kwok_cluster_<name>`). To reset a cluster, use a new `name`.
- **Custom images** must contain `kwokctl` in `PATH`; the startup script fails fast with an actionable message otherwise.
- **CIDR params** (`clusterCidr`, `serviceCird`) are k3s-parity passthroughs wired via `kwokctl --extra-args` (verified with the default image, kwokctl v0.6.1). They rarely matter for kwok (no real pods/CNI); note that kwokctl prepends `--` to extra-args itself, so a custom `image` with a different kwok version may parse them differently.
- **`config` before `server`** returns a clean error (no panic): `ERROR: kubeconfig not found - start the cluster first with 'server'`.
- **Only one `server` instance per cluster name**: a second `server up` for the same name fails fast with `already being served by another service instance`. The imperative functions (`apply`, `wait-for-deployment-ready`, `simulate-rolling-update`) detect an already-running cluster and reuse it instead of starting a second instance, so they can be used from another shell while `server up` is running.

## Security considerations

This module is a **development/testing tool** — never use it in production or in multi-tenant environments. Its security posture (largely k3s parity):

- **Admin credentials by design**: the kubeconfig returned by `config` embeds the cluster CA and the **cluster-admin** client certificate/key (embedded as data so it is portable). Anyone holding that file fully controls the (simulated) cluster. Treat exported kubeconfigs as secrets and do not publish port 6443 (`server up`) on untrusted networks. The API itself is secure-only (TLS on 6443, the insecure port is disabled) and requires the client certificate.
- **Root & capabilities (k3s parity)**: the cluster service runs as root with `InsecureRootCapabilities: true`, which is required for cgroup nesting and for running etcd/kube-apiserver inside the container. The `kubectl`/`k9s` helper containers run as non-root user `1001`.
- **Cache volume holds the PKI**: cluster state (etcd data, PKI private keys, kubeconfig) persists in the Dagger cache volume `kwok_cluster_<name>`. Functions that mount it can read the PKI. Use a new `name` to reset/discard state.
- **Input validation**: cluster names, deployment names and namespaces are validated as DNS-1123 labels, image references against a strict allowlist, and CIDRs with `net.ParseCIDR` before they are interpolated into shell scripts, kubectl argument lists or YAML manifests. All internal kubectl invocations use exec form (no shell). The single exception is `kubectl --args`, which is shell-interpreted on purpose (k3s parity) — do not pass untrusted input there.
- **Resource bounds**: `nodes` and `replicas` are capped (`1000` each), API-server readiness and deployment waits are deadline-bounded, and kubectl polls use `--request-timeout`, so the module cannot be made to wait or allocate without bound.
- **Helper images**: `alpine`, `bitnami/kubectl` and `derailed/k9s` are used without pinned tags (k3s parity). If your threat model requires supply-chain reproducibility, pin them by digest in a fork.
- **Simulated workloads**: pods never run real containers; manifests applied via `apply` only affect the disposable simulated cluster.
