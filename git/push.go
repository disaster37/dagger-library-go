package git

import (
	"context"

	"dagger.io/dagger"
)

func CommitAndPush(ctx context.Context, client *dagger.Client, path string) (err error) {

	_, err = getGitContainer(client, path).
		WithEntrypoint([]string{"/bin/sh", "-c"}).
		WithExec([]string{`
set -e

git add -A
git commit -m "push back from pipeline"
git push
		`}).
		Stdout(ctx)

	return err

}
