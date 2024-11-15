package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"

	"github.com/disaster37/dagger-library-go/lib/helper"
)

type TestResult struct {
	// +private
	Base *dagger.Container
}

func NewTestResult(container *dagger.Container) *TestResult {
	return &TestResult{
		Base: container,
	}
}

func (h *TestResult) Stdout(ctx context.Context) (string, error) {
	return h.Base.Stdout(ctx)
}

func (h *TestResult) Coverage() *dagger.File {
	return h.Base.
		WithExec(helper.ForgeScript(`cat coverage.out.tmp | grep -v "_generated.*.go" > coverage.out`)).
		File("coverage.out")
}
