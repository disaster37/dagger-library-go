# dagger

## CI pipeline

It will run the following steps:
  - Call `ci` function

### 1. Minimal execution (no credentials)

```bash
dagger call -m 'github.com/disaster37/dagger-library-go/helm@v2' --src '.' 'ci' --version env:VERSION --git-repo-url env:GIT_REPO_URL --git-branch env:BRANCH export --path .
```

### 2. With credentials

```bash
dagger call -m 'github.com/disaster37/dagger-library-go/helm@v2' --src '.' 'ci' --version env:VERSION --git-repo-url env:GIT_REPO_URL --git-branch env:BRANCH --registry-username env:SU_USERNAME --registry-password env:SU_PASSWORD export --path .
```

### 3. Full CI execution

```bash
dagger call -m 'github.com/disaster37/dagger-library-go/helm@v2' --src '.' 'ci' --version env:VERSION --git-repo-url env:GIT_REPO_URL --git-branch env:BRANCH --registry-username env:SU_USERNAME --registry-password env:SU_PASSWORD --ci github export --path .
```

## Extra section

Some extra content.
