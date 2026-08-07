# Image module

Dagger module for Docker image lifecycle: lint, build, push, and CI pipeline generation.

## Local test

For local testing:

```bash
dagger call -m dagger/image lint --source . --dockerfile fixtures/Dockerfile
dagger call -m dagger/image build --source . --dockerfile fixtures/Dockerfile push --repository-name disaster37/test --registry-url ttl.sh --version 1m
```

## Functions

### `Ci` — Full CI pipeline

Runs lint and build as a single CI entrypoint.

| Arg | Default | Description |
|-----|---------|-------------|
| `source` | (required) | Source directory |
| `dockerfile` | `Dockerfile` | Dockerfile path |

### `GenerateCi` — Generate CI pipeline files

Generates a CI pipeline file for GitHub Actions, Jenkins, or GitLab CI.

| Arg | Default | Description |
|-----|---------|-------------|
| `ci` | (required) | `github`, `jenkins`, or `gitlab` |
| `branches` | `["main"]` | Branches that trigger the pipeline |
| `daggerVersion` | current | Dagger CLI version |
| `registry` | auto | OCI registry URL (GitHub default: `ghcr.io`) |
| `repository` | auto | Repository path (GitHub default: `${{ github.repository }}`) |
| `defaultBranch` | `main` | Branch used for tag commits |
| `moduleRef` | auto (current version) | Configurable module reference |
| `registryUsernameKey` | — | GitHub secret name for username |
| `registryPasswordKey` | — | GitHub secret name for password |
| `registryCredential` | — | Jenkins credential ID for registry |
| `gitTokenCredential` | — | Jenkins credential ID for git token |
| `registryUsernameVar` | — | GitLab CI/CD variable name for registry username |
| `registryPasswordVar` | — | GitLab CI/CD variable name for registry password |
| `gitTokenVar` | — | GitLab CI/CD variable name for git token |

```bash
# Generate GitHub Actions workflow
dagger call -m dagger/image generate-ci --ci github --registry ghcr.io --repository myorg/myimage export --path .

# Generate Jenkins pipeline
dagger call -m dagger/image generate-ci --ci jenkins --registry ghcr.io --repository myorg/myimage --registry-credential my-creds --git-token-credential my-token export --path .

# Generate GitLab CI pipeline
dagger call -m dagger/image generate-ci --ci gitlab --registry ghcr.io --repository myorg/myimage --registry-username-var REGISTRY_USER --registry-password-var REGISTRY_PASS --git-token-var GIT_TOKEN export --path .
```
