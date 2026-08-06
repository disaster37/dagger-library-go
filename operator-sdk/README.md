# Operator SDK module

Dagger module for Operator SDK lifecycle: generate manifests/bundle, build images, publish to OCI registries, test on K3s, and CI pipeline generation.

## Functions

### `Release`

Full operator release: generate manifests, build operator/bundle/catalog images, publish to OCI registry.

### `GenerateCi` — Generate CI pipeline files

Generates a CI pipeline file for GitHub Actions, Jenkins, or GitLab CI.

| Arg | Default | Description |
|-----|---------|-------------|
| `ci` | (required) | `github`, `jenkins`, or `gitlab` |
| `branches` | `["main"]` | Branches that trigger the pipeline |
| `daggerVersion` | current | Dagger CLI version |
| `registry` | auto | OCI registry URL (GitHub default: `ghcr.io`) |
| `repository` | auto | Repository path |
| `defaultBranch` | `main` | Branch used for tag commits |
| `moduleRef` | `github.com/disaster37/dagger-library-go/operator-sdk@v2` | Configurable module reference |
| `registryUsernameKey` | — | GitHub secret name for username |
| `registryPasswordKey` | — | GitHub secret name for password |
| `registryCredential` | — | Jenkins credential ID for registry |
| `gitTokenCredential` | — | Jenkins credential ID for git token |
| `registryUsernameVar` | — | GitLab CI/CD variable name for registry username |
| `registryPasswordVar` | — | GitLab CI/CD variable name for registry password |
| `gitTokenVar` | — | GitLab CI/CD variable name for git token |

```bash
# Generate GitHub Actions workflow
dagger call -m dagger/operator-sdk generate-ci --ci github --registry ghcr.io --repository myorg/myoperator export --path .

# Generate Jenkins pipeline
dagger call -m dagger/operator-sdk generate-ci --ci jenkins --registry ghcr.io --repository myorg/myoperator --registry-credential my-creds --git-token-credential my-token export --path .

# Generate GitLab CI pipeline
dagger call -m dagger/operator-sdk generate-ci --ci gitlab --registry ghcr.io --repository myorg/myoperator --registry-username-var REGISTRY_USER --registry-password-var REGISTRY_PASS --git-token-var GIT_TOKEN export --path .
```
