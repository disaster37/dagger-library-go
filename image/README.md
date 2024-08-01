# Image module

This module permit to build image from Dockerfile and the push it on registry

## Local test

For local test purpose

```bash
dagger call build --source . --dockerfile fixtures/Dockerfile  push --repository-name disaster37/test --registry-url ttl.sh --version 1m
```