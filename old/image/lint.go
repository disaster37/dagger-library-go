package image

import (
	"context"
	"fmt"

	"dagger.io/dagger"
	"emperror.dev/errors"
	"github.com/creasty/defaults"
	"github.com/disaster37/dagger-library-go/helper"
	"github.com/gookit/validate"
)

type LintOption struct {
	PathContext string `default:"."`
	Dockerfile  string `default:"Dockerfile"`
}

// Lint permit to lint helm
func Lint(ctx context.Context, client *dagger.Client, option *LintOption) (err error) {

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	_, err = getHadolintContainer(client, option.PathContext).
		WithExec(helper.ForgeCommand(fmt.Sprintf("/bin/hadolint --failure-threshold error %s", option.Dockerfile))).
		Stdout(ctx)
	if err != nil {
		return errors.Wrap(err, "Error when lint Dockerfile")
	}

	return nil
}
