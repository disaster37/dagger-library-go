package main

import (
	"context"

	"emperror.dev/errors"
	"gopkg.in/yaml.v3"
)

// PushUrl returns the OCI chart URL that would be pushed.
// This is useful as a summary/report at the end of a CI run.
func (m *Helm) PushUrl(
	ctx context.Context,

	// The registry url
	registryUrl string,

	// The repository name
	repositoryName string,

	// The version
	version string,

) (string, error) {

	chartContents, err := m.Src.File("Chart.yaml").Contents(ctx)
	if err != nil {
		return "", errors.Wrap(err, "Error when read Chart.yaml")
	}
	dataChart := make(map[string]any)
	if err := yaml.Unmarshal([]byte(chartContents), &dataChart); err != nil {
		return "", errors.Wrap(err, "Error when decode YAML file")
	}
	chartName := dataChart["name"].(string)

	return "oci://" + registryUrl + "/" + repositoryName + "/" + chartName + ":" + version, nil
}
