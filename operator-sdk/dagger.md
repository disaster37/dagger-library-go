## Description

This module provides CI/CD functions for Operator SDK projects:

- **Build**: Build operator image
- **Test**: Run operator tests against a k3s cluster
- **Bundle**: Generate and validate OLM bundle
- **Catalog**: Generate OLM catalog and push to registry
- **Release**: Full release pipeline (build, test, bundle, catalog, push)

## Usage

```bash
dagger call -m github.com/disaster37/dagger-library-go/operator-sdk build --src .
dagger call -m github.com/disaster37/dagger-library-go/operator-sdk release --src . --version 1.0.0 ...
```
