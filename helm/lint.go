package main

import (
	"context"

	"github.com/disaster37/dagger-library-go/lib/v2/helper"
)

// Lint permit to lint helm chart
func (m *Helm) Lint(
	ctx context.Context,
) (stdout string, err error) {

	return m.HelmContainer.
		WithExec(helper.ForgeCommand("helm dependency update")).
		WithExec(helper.ForgeCommand("helm lint .")).
		Stdout(ctx)
}
