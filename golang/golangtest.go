package main

import (
	"context"
	"dagger/golang/internal/dagger"

	"github.com/disaster37/dagger-library-go/lib/helper"
)

type GolangTest struct {
	// The base container
	Container *dagger.Container
}

// NewGolangTest permit to init new Golang test result
func NewGolangTest(container *dagger.Container) *GolangTest {
	return &GolangTest{
		Container: container,
	}
}

// Stdout to display the stdout of tests
func (h *GolangTest) Stdout(ctx context.Context) (string, error) {
	return h.Container.Stdout(ctx)
}

func (h *GolangTest) Coverage(ctx context.Context) *dagger.File {
	return h.Container.
		WithExec(helper.ForgeScript(`cat coverage.out.tmp | grep -v "_generated.*.go" > coverage.out`)).
		File("coverage.out")
}
