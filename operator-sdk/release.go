package main

import (
	"context"
	"dagger/operator-sdk/internal/dagger"
)

type Release struct {
	// +private
	Std string

	// +private
	Dir *dagger.Directory
}

func NewRelease(stdout string, dir *dagger.Directory) *Release {
	return &Release{
		Std: stdout,
		Dir: dir,
	}
}

func (h *Release) Stdout(
	ctx context.Context,
) string {
	return h.Std
}

func (h *Release) Directory() *dagger.Directory {
	return h.Dir
}
