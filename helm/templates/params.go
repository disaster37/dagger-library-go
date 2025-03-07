package templates

type Opts struct {
	DaggerVersion              string
	Registry                   string
	Repository                 string
	RegistryCredential         string
	GitTokenCredential         string
	RegistryUsernameSecretName string
	RegistryPasswordSecretName string
	DefaultBranchName          string
}
