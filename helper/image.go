package helper

import (
	"os"

	"dagger.io/dagger"
)

func WithProxy(c *dagger.Container) *dagger.Container {
	return c.WithEnvVariable("HTTP_PROXY", os.Getenv("HTTP_PROXY")).
		WithEnvVariable("HTTPS_PROXY", os.Getenv("HTTPS_PROXY")).
		WithEnvVariable("NO_PROXY", os.Getenv("NO_PROXY"))
}
