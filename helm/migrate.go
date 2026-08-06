package main

import (
	"context"
	"dagger/helm/internal/dagger"

	"github.com/disaster37/dagger-library-go/lib/helper"
)

// MigrateValuesTags permit to migrate values.yaml metadata tags
// It rewrites old concrete array indices ([0], [1]) in @param / @skip / @extra
// metadata comment paths to the new generic [] syntax used by the Go
// readme-generator-for-helm fork. The operation is idempotent and only modifies
// metadata comment lines; YAML values, modifiers, and descriptions are preserved.
// It returns the migrated values file.
func (m *Helm) MigrateValuesTags(
	ctx context.Context,

	// The values file to migrate
	// +optional
	// +default="values.yaml"
	valuesFile string,

	// Config file for readme-generator
	// +optional
	configFile string,

	// Output file path. If empty, valuesFile is rewritten in place.
	// +optional
	outputFile string,
) (migratedFile *dagger.File, err error) {

	container := m.GeneratorContainer

	if valuesFile == "" {
		valuesFile = "values.yaml"
	}

	if err = validatePathArg("valuesFile", valuesFile); err != nil {
		return nil, err
	}
	if err = validatePathArg("configFile", configFile); err != nil {
		return nil, err
	}
	if err = validatePathArg("outputFile", outputFile); err != nil {
		return nil, err
	}

	cmd := "readme-generator migrate" + configArg(configFile) + " -v " + valuesFile
	if outputFile != "" {
		cmd += " --output " + outputFile
	}
	container = container.WithExec(helper.ForgeCommand(cmd))

	effectivePath := valuesFile
	if outputFile != "" {
		effectivePath = outputFile
	}
	migratedFile = container.File(effectivePath)

	return migratedFile, nil
}
