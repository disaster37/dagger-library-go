package helm

type HelmOption struct {
	WithProxy            bool   `default:"true"`
	WithPush             bool   `default:"false"`
	WithRegistryUsername string `validate:"validateRegistryAuth"`
	WithRegistryPassword string `validate:"validateRegistryAuth"`
	RegistryUrl          string `validate:"validateRegistryAuth"`
	RepositoryName       string `validate:"validateRegistryAuth"`
	PathContext          string `default:"."`
	CaPath               string
	Version              string
	SchemaFileName       string
	ReadmeFileName       string
}

func (h HelmOption) ValidateRegistryAuth(val string) bool {
	if h.WithPush && val == "" {
		return false
	}

	return true
}
