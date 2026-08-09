package uploadcmd

import (
	"slices"
	"strings"
	"testing"
)

func TestBuild(t *testing.T) {
	cases := []struct {
		name    string
		inName  string
		inFiles []string
		inFlags []string
		want    []string
	}{
		{
			name: "no optionals",
			want: []string{"/bin/codecov", "-v"},
		},
		{
			name:   "name set",
			inName: "unit",
			want:   []string{"/bin/codecov", "-v", "-n", "unit"},
		},
		{
			name:   "name with space stays one argv element",
			inName: "my upload",
			want:   []string{"/bin/codecov", "-v", "-n", "my upload"},
		},
		{
			name:    "files with multiple entries",
			inFiles: []string{"a.out", "b.out"},
			want:    []string{"/bin/codecov", "-v", "-f", "a.out", "b.out"},
		},
		{
			name:    "flags appended verbatim",
			inFlags: []string{"-F", "unit", "--debug"},
			want:    []string{"/bin/codecov", "-v", "-F", "unit", "--debug"},
		},
		{
			name:    "flag with space stays one argv element",
			inFlags: []string{"--foo=bar baz"},
			want:    []string{"/bin/codecov", "-v", "--foo=bar baz"},
		},
		{
			name:    "all three optionals combined",
			inName:  "unit",
			inFiles: []string{"a.out", "b.out"},
			inFlags: []string{"-F", "unit", "--debug"},
			want:    []string{"/bin/codecov", "-v", "-n", "unit", "-f", "a.out", "b.out", "-F", "unit", "--debug"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Build(c.inName, c.inFiles, c.inFlags)
			if !slices.Equal(got, c.want) {
				t.Errorf("Build(%q, %v, %v) = %v, want %v",
					c.inName, c.inFiles, c.inFlags, got, c.want)
			}
		})
	}
}

func TestBuild_NoTokenInArgv(t *testing.T) {
	cmd := Build("unit", []string{"a.out", "b.out"}, []string{"-F", "unit", "--debug"})

	for _, arg := range cmd {
		if arg == "-t" {
			t.Fatalf("token flag -t must never appear in argv, got %v", cmd)
		}
		if strings.Contains(arg, "CODECOV_TOKEN") {
			t.Fatalf("CODECOV_TOKEN must never appear in argv, got %v", cmd)
		}
		if strings.Contains(arg, "$") {
			t.Fatalf("argv must contain no shell expansion ($), got %v", cmd)
		}
	}
}
