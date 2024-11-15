# Contribute

PR are alway welcome here.

## Local test

### Test SDK sub module

**Display current version**:
```bash
dagger call -m "./operator-sdk" sdk version
```

**Init operator sdk project**:

```bash
dagger call --src /tmp/test sdk run --cmd "init --domain example.com --repo github.com/example/memcached-operator" export --path /tmp/test
```

