// A generated module for CodeWorkspace functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/code-workspace/internal/dagger"
	"strings"

	"emperror.dev/errors"
)

// Interface for something that can be checked
type Checkable interface {
	dagger.DaggerObject
	LintProject(ctx context.Context, source *dagger.Directory) (string, error)
	RunTest(ctx context.Context, source *dagger.Directory) (string, error)
	RunVulnCheck(ctx context.Context, source *dagger.Directory) (string, error)
	FormatProject(source *dagger.Directory) *dagger.Directory
	BuildProject(ctx context.Context, source *dagger.Directory) (string, error)
}

type CodeWorkspace struct {
	Work *dagger.Directory
	// +private
	Start *dagger.Directory
	// +private
	Checker Checkable
}

func New(
	// Initial state of the workspace
	source *dagger.Directory,
	// Checker to use for testing
	checker Checkable,
) *CodeWorkspace {
	return &CodeWorkspace{
		Start:   source,
		Work:    source,
		Checker: checker,
	}
}

// Read the contents of a file in the workspace at the given path
func (w *CodeWorkspace) Read(
	ctx context.Context,
	// Path to read the file at
	path string,
) (string, error) {
	return w.Work.File(path).Contents(ctx)
}

// Write the contents of a file in the workspace at the given path
func (w *CodeWorkspace) Write(
	ctx context.Context,
	// Path to write the file to
	path string,
	// Contents to write to the file
	contents string,
) *CodeWorkspace {
	// Write new file
	w.Work = w.Work.WithNewFile(path, contents)

	return w
}

// Reset the workspace to the original state
func (w *CodeWorkspace) Reset() *CodeWorkspace {
	w.Work = w.Start
	return w
}

// List the files in the workspace in tree format
func (w *CodeWorkspace) Tree(ctx context.Context) (string, error) {
	return dag.Container().From("alpine:3").
		WithDirectory("/workspace", w.Work).
		WithExec([]string{"tree", "/workspace"}).
		Stdout(ctx)
}

// Lint code, then run vuln check, then run tests
func (w *CodeWorkspace) Check(ctx context.Context) (string, error) {
	var (
		stdo string
		err  error
		sb   strings.Builder
	)

	// lint code
	stdo, err = w.Checker.LintProject(ctx, w.Work)
	if err != nil {
		return stdo, errors.Wrap(err, "error when lint project")
	}
	sb.WriteString("Lint code stdout:\n")
	sb.WriteString(stdo)
	sb.WriteString("\n")

	// Run VulnCheck
	stdo, err = w.Checker.RunVulnCheck(ctx, w.Work)
	if err != nil {
		return stdo, errors.Wrap(err, "error when run vuln check")
	}
	sb.WriteString("Vuln check stdout:\n")
	sb.WriteString(stdo)
	sb.WriteString("\n")

	// Run tests
	stdo, err = w.Checker.RunTest(ctx, w.Work)
	if err != nil {
		return stdo, errors.Wrap(err, "error when run tests")
	}
	sb.WriteString("Tests stdout:\n")
	sb.WriteString(stdo)
	sb.WriteString("\n")

	return sb.String(), nil

}

// Show the changes made to the workspace so far in unified diff format
func (w *CodeWorkspace) Diff(ctx context.Context) (string, error) {
	return dag.Container().From("alpine:3").
		WithDirectory("/a", w.Start).
		WithDirectory("/b", w.Work).
		WithExec([]string{"diff", "-rN", "a/", "b/"}, dagger.ContainerWithExecOpts{Expect: dagger.ReturnTypeAny}).
		Stdout(ctx)
}

// Format wil format all files on project as expected
func (w *CodeWorkspace) Format(ctx context.Context) *CodeWorkspace {
	w.Work.WithDirectory(".", w.Checker.FormatProject(w.Work))

	return w
}
