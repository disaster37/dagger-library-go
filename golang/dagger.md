## Description

This module provides CI/CD functions for Go projects:

- **Build**: Cross-compile Go binaries for multiple OS/arch
- **Test**: Run tests with coverage output
- **Lint**: Run golangci-lint
- **Format**: Format Go source code
- **Vulncheck**: Scan for known vulnerabilities with govulncheck

## Usage

**All the pipeline (lint, vulncheck, test, format, build)**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/golang@2.0.18' --src . ci
```

**Build only**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/golang@2.0.18' --src . build
```

**Test only**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/golang@2.0.18' --src . test
```

**Lint only**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/golang@2.0.18' --src . lint
```

**Format**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/golang@2.0.18' --src . format
```

**Vuln check**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/golang@2.0.18' --src . vulncheck
```

**Bench**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/golang@2.0.18' --src . bench
```

**Debug**
```bash
dagger call -m 'github.com/disaster37/dagger-library-go/golang@2.0.18' --src . debug up
```