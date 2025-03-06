package ci

type CI string

const (
	// Github CI
	Github CI = "github"

	// Jenkins CI
	Jenkins CI = "jenkins"

	// Default CI
	Dagger CI = "dagger"
)
