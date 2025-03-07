package templates

const module = "github.com/disaster37/dagger-library-go/helm@v2"

type Opts struct {
	DaggerVersion              string
	Registry                   string
	Repository                 string
	RegistryCredential         string
	GitTokenCredential         string
	RegistryUsernameSecretName string
	RegistryPasswordSecretName string
	DefaultBranchName          string
	HelmPathOpt                string
}
