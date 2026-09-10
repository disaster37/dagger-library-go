## Description

This module provides CI/CD functions for Go projects:

- **Build**: Cross-compile Go binaries for multiple OS/arch
- **Test**: Run tests with coverage output
- **Lint**: Run golangci-lint
- **Format**: Format Go source code
- **Vulncheck**: Scan for known vulnerabilities with govulncheck

## Usage

```bash
dagger call -m github.com/disaster37/dagger-library-go/golang build --src .
dagger call -m github.com/disaster37/dagger-library-go/golang test --src .
```
