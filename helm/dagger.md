## Description

This module provides CI/CD functions for Helm charts:

- **Lint**: Validate Helm chart structure and templates
- **Schema**: Generate JSON schema from `values.yaml`
- **Documentation**: Generate README from chart metadata and values
- **Push**: Package and push chart to OCI registry
- **Release**: Lint, schema, doc, push, then commit/push changes to git

## Usage

**Full pipeline (lint, generate doc, generate schema) when no need auth on dependency repository**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/helm@2.0.18' --src . ci --verison 0.0.1 export --path .
```

**Full pipeline (lint, generate doc, generate schema)  when required auth on dependency repository**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/helm@2.0.18' --src . with-extra-chart-repository --name my-repo --url 'oci://my-repo.local.domain' --username 'env:user' --password 'env:password' ci --verison 0.0.1 export --path .
```

**Lint only (without auth on dependency repository)**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/helm@2.0.18' --src . lint
```

**Lint only (with auth on dependency repository)**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/helm@2.0.18' --src . with-extra-chart-repository --name my-repo --url 'oci://my-repo.local.domain' --username 'env:user' --password 'env:password' lint
```

**Generate parameters on README.md**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/helm@2.0.18' --src . generate-documentation export --path .
```

**Generate shema validation**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/helm@2.0.18' --src . generate-schema export --path .
```