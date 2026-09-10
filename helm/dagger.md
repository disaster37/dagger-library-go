## Description

This module provides CI/CD functions for Helm charts:

- **Lint**: Validate Helm chart structure and templates
- **Schema**: Generate JSON schema from `values.yaml`
- **Documentation**: Generate README from chart metadata and values
- **Push**: Package and push chart to OCI registry
- **Release**: Lint, schema, doc, push, then commit/push changes to git

## Usage

```bash
dagger call -m github.com/disaster37/dagger-library-go/helm lint --src .
dagger call -m github.com/disaster37/dagger-library-go/helm release --src . --version 1.0.0 ...
```
