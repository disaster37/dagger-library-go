package helper

import (
	"os"

	"dagger.io/dagger"
)

func WithProxy(c *dagger.Container) *dagger.Container {
	if os.Getenv("HTTP_PROXY") != "" {
		c = c.WithEnvVariable("HTTP_PROXY", os.Getenv("HTTP_PROXY"))
	}
	if os.Getenv("HTTPS_PROXY") != "" {
		c = c.WithEnvVariable("HTTPS_PROXY", os.Getenv("HTTPS_PROXY"))
	}
	if os.Getenv("NO_PROXY") != "" {
		c = c.WithEnvVariable("NO_PROXY", os.Getenv("NO_PROXY"))
	}
	if os.Getenv("http_proxy") != "" {
		c = c.WithEnvVariable("HTTP_PROXY", os.Getenv("http_proxy"))
	}
	if os.Getenv("https_proxy") != "" {
		c = c.WithEnvVariable("HTTPS_PROXY", os.Getenv("https_proxy"))
	}
	if os.Getenv("no_proxy") != "" {
		c = c.WithEnvVariable("NO_PROXY", os.Getenv("no_proxy"))
	}

	return c
}
