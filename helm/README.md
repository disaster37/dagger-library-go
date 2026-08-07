# Helm module

Dagger module for Helm chart lifecycle: lint, schema/docs generation, version bump, OCI push, and CI pipeline generation (GitHub Actions / Jenkins / GitLab CI).

## Quick start

```bash
dagger call -m github.com/disaster37/dagger-library-go/helm@2.0.4 --src . ci export --path .
```

## Functions

### `New` — Initialize

| Arg | Required | Default | Description |
|-----|----------|---------|-------------|
| `src` | yes | — | Helm chart source directory |
| `baseHelmContainer` | no | `alpine/helm:3.14.3` | Custom container with `helm` |
| `baseGeneratorContainer` | no | `readme-generator-for-helm:0.0.3` | Custom container with `readme-generator` |
| `baseYqContainer` | no | `mikefarah/yq:4.35.2` | Custom container with `yq` |

### `WithSource(src)` / `WithWorkDir(path)`

Change the source directory or working directory on all internal containers.

### `WithRepository`

Authenticate against a private Helm registry (OCI or classic).

```bash
dagger call -m helm --src . with-repository \
  --name myrepo --url https://charts.example.com \
  --username env:MY_USER --password env:MY_PASS \
  lint
```

| Arg | Required | Description |
|-----|----------|-------------|
| `name` | no (for OCI) | Repository name |
| `url` | yes | Repository URL |
| `isOci` | no (default `false`) | Use OCI registry login |
| `username` | yes | Registry username secret |
| `password` | yes | Registry password secret |

### `Lint`

Lint the chart (runs `helm dependency update` + `helm lint .`).

```bash
dagger call -m helm --src /path/to/chart lint
```

### `GenerateSchema`

Generate `values.schema.json` from `values.yaml` using `readme-generator-for-helm`.

| Arg | Default | Description |
|-----|---------|-------------|
| `targetFile` | `values.schema.json` | Output file name |
| `configFile` | — | Generator config file |

### `GenerateDocumentation`

Generate `README.md` parameter table from `values.yaml`.

| Arg | Default | Description |
|-----|---------|-------------|
| `targetFile` | `README.md` | Output file name |
| `configFile` | — | Generator config file |

### `MigrateValuesTags`

Rewrite legacy array index syntax in values.yaml comments (`[0]` → `[]`).

| Arg | Default | Description |
|-----|---------|-------------|
| `valuesFile` | `values.yaml` | File to migrate |
| `configFile` | — | Generator config file |
| `outputFile` | — | Output path (empty = rewrite in place) |

### `UpdateChart` / `UpdateValues`

Set a YAML key in `Chart.yaml` or `values.yaml` using `yq`.

```bash
dagger call -m helm --src . update-chart --key .version --value 1.2.3
```

### `Push`

Package and push a chart to an OCI registry. Returns the updated `Chart.yaml`.

| Arg | Required | Description |
|-----|----------|-------------|
| `registryUrl` | yes | OCI registry URL |
| `repositoryName` | yes | Repository path inside the registry |
| `version` | yes | Chart version to publish |

### `Ci` — Full CI pipeline

Orchestrates the complete release workflow:

1. Authenticate to registry (if `--registry` is set)
2. For each `--helm-paths`: generate schema, generate docs, lint, push
3. Commit & push changes back (if `--ci` is set and `--dry-run` is not set)

| Arg | Required (CI mode) | Description |
|-----|--------------------|-------------|
| `registry` | yes | OCI registry URL |
| `repository` | yes | Repository name |
| `helmPaths` | no (default `["."]`) | Chart directories |
| `ci` | yes | CI type: `github`, `jenkins`, or `gitlab` |
| `version` | yes | Semantic version to publish |
| `registryUsername` | yes | Registry username secret |
| `registryPassword` | yes | Registry password secret |
| `gitToken` | yes | Git token secret |
| `gitRepoUrl` | yes | Git repository URL |
| `gitBranch` | yes (PR/tag) | Target branch |
| `dryRun` | no (default `false`) | Skip push and git commit/push |

Version handling: if `version` has a prerelease suffix (e.g. `0.0.0-rc`), it bumps the chart's current patch version and appends the prerelease.

### `GenerateCi` — Generate CI pipeline files

Generates a CI pipeline file and a `DAGGER.md` usage guide.

| Arg | Default | Description |
|-----|---------|-------------|
| `ci` | (required) | `github`, `jenkins`, or `gitlab` |
| `branches` | `["main"]` | Branches that trigger the pipeline |
| `helmPaths` | `["."]` | Chart directory paths |
| `daggerVersion` | current | Dagger CLI version |
| `registry` | auto | OCI registry URL (GitHub default: `ghcr.io`) |
| `repository` | auto | Repository path (GitHub default: `${{ github.repository }}`) |
| `defaultBranch` | `main` | Branch used for tag commits |
| `moduleRef` | auto (current version) | Configurable dagger module reference |
| `registryCredential` | — | Jenkins credential ID for registry |
| `registryUsernameKey` | — | GitHub secret name for username |
| `registryPasswordKey` | — | GitHub secret name for password |
| `gitTokenCredential` | — | Jenkins credential ID for git token |
| `registryUsernameVar` | — | GitLab CI/CD variable name for registry username |
| `registryPasswordVar` | — | GitLab CI/CD variable name for registry password |
| `gitTokenVar` | — | GitLab CI/CD variable name for git token |

## Generating CI pipelines

### GitHub Actions

```bash
dagger call -m github.com/disaster37/dagger-library-go/helm@2.0.4 \
  --src . generate-ci --ci github \
  --branches main --branches develop \
  --helm-paths charts/myapp \
  export --path .
```

This creates:
- `.github/workflows/dagger.yaml` — triggers on push to configured branches, tags, PRs, and releases
- `DAGGER.md` — local usage instructions

**Pipeline behavior:**
- On **push to branch**: publishes `0.0.0-rc.{run_number}`, commits to the branch
- On **tag**: publishes the tag as version, commits to `defaultBranch` (typically `main`)
- On **pull request**: publishes `0.0.0-pr.{pr_number}.{run_number}`, pushes to the PR branch
- On **release**: triggers the same pipeline

**Override module ref:**
```bash
dagger call -m helm --src . generate-ci --ci github --module-ref github.com/myorg/helm@v3 export --path .
```

**Required GitHub secrets:**
- `GITHUB_TOKEN` — auto-provided, needs `contents: write` and `packages: write` permissions
- `DAGGER_CLOUD_TOKEN` — Dagger Cloud token (optional but recommended)

### Jenkins

```bash
dagger call -m github.com/disaster37/dagger-library-go/helm@2.0.4 \
  --src . generate-ci --ci jenkins \
  --branches main \
  --helm-paths charts/myapp \
  --registry ghcr.io --repository myorg/helm-charts \
  --registry-credential my-registry-creds \
  --git-token-credential my-git-token \
  export --path .
```

This creates a `Jenkinsfile` at the repository root.

**Requirements:**
- A Jenkins Kubernetes agent template named `dagger` with the Dagger CLI installed
- Credentials configured in Jenkins:
  - `registryCredential` — username/password credential for the OCI registry
  - `gitTokenCredential` — secret text credential for git push

### GitLab CI

```bash
dagger call -m github.com/disaster37/dagger-library-go/helm@2.0.4 \
  --src . generate-ci --ci gitlab \
  --branches main \
  --helm-paths charts/myapp \
  --registry ghcr.io --repository myorg/helm-charts \
  --registry-username-var REGISTRY_USERNAME \
  --registry-password-var REGISTRY_PASSWORD \
  --git-token-var GIT_TOKEN \
  export --path .
```

This creates a `.gitlab-ci.yml` at the repository root.

**Requirements:**
- GitLab CI/CD variables configured:
  - `REGISTRY_USERNAME` — registry username
  - `REGISTRY_PASSWORD` — registry password
  - `GIT_TOKEN` — git access token

### Local execution (dry-run)

With `--ci` omitted, `ci` skips push and git commit. Use `--dry-run` to skip push/commit even when `--ci` is set:

```bash
dagger call -m github.com/disaster37/dagger-library-go/helm@2.0.4 --src . ci --ci github --dry-run export --path .
```

This runs all steps (lint, schema, docs) but skips registry push and git commit/push.

## Design notes

The CI pipeline generator lives in `lib/pipeline` as a pure Go library (no Dagger SDK imports), making it fully unit-testable without a Dagger engine. The library programmatically builds CI pipelines for GitHub Actions, Jenkins, and GitLab CI, replacing the previous quicktemplate-based approach.

Key improvements:
- **Testable**: golden-file tests compare render output; `go test ./pipeline/` runs without a Dagger engine
- **Configurable module ref**: `--module-ref` overrides the module path in generated pipelines
- **GitLab CI support**: new renderer for `.gitlab-ci.yml`
- **Dry-run mode**: `--dry-run` flag on `ci` skips push and git commit
- **Reusable**: `lib/pipeline` is imported by all Dagger modules that need CI generation
