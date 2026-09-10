## Description

This module provides CI/CD functions for Docker images:

- **Build**: Build Docker image from Dockerfile
- **Lint**: Lint Dockerfile with hadolint
- **Push**: Push image to OCI-compatible registry
- **Scan**: Scan image for vulnerabilities with trivy

## Usage

```bash
dagger call -m github.com/disaster37/dagger-library-go/image build --source . --dockerfile Dockerfile
dagger call -m github.com/disaster37/dagger-library-go/image ci --source . --ci=true --registry ghcr.io ...
```
