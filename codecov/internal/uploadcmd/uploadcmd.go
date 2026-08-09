// Package uploadcmd assembles the codecov uploader command line.
// It deliberately has no Dagger dependency so it can be unit-tested
// without a Dagger session.
package uploadcmd

// Build returns the argv for the codecov uploader.
//
// The token must NEVER be part of argv: it is provided to the container as
// the CODECOV_TOKEN secret environment variable, which the uploader reads
// automatically. Embedding it in argv would expose it in process listings
// and Dagger logs, and historically it was corrupted by shell expansion.
func Build(name string, files []string, flags []string) []string {
	cmd := []string{"/bin/codecov", "-v"}

	if name != "" {
		cmd = append(cmd, "-n", name)
	}

	if len(files) > 0 {
		cmd = append(cmd, "-f")
		cmd = append(cmd, files...)
	}

	// Raw passthrough: each element becomes exactly one argv entry.
	return append(cmd, flags...)
}
