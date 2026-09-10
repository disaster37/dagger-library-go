## Description

This module provides CI/CD functions for Docker images:

- **Build**: Build Docker image from Dockerfile
- **Lint**: Lint Dockerfile with hadolint
- **Push**: Push image to OCI-compatible registry
- **Scan**: Scan image for vulnerabilities with trivy

## Usage

**All ci (lint, build)
```bash
dagger call -m github.com/disaster37/dagger-library-go/image@2.0.18 ci --source .
```

** All ci (lint, build) with build args
```bash
dagger call -m github.com/disaster37/dagger-library-go/image@2.0.18 with-build-arg --name arg1 --val value1 ci --source .
```

**Lint only**
```bash
dagger call -m github.com/disaster37/dagger-library-go/image@2.0.18 lint --source .
```

**Build only**
```bash
dagger call -m github.com/disaster37/dagger-library-go/image@2.0.18 build --source .
```

**Build only with build arg**
```bash
dagger call -m github.com/disaster37/dagger-library-go/image@2.0.18 with-build-arg --name arg1 --val value1 build --source .
```