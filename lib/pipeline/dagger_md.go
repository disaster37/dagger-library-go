package pipeline

import (
	"fmt"
	"strings"
)

// renderLocalCommand builds the `dagger call` shell command with
// env:VAR placeholders for local usage.
func renderLocalCommand(spec PipelineSpec) string {
	cmd, _, _ := assembleShellCommand(spec)
	localCmd := cmd
	for ph, binding := range spec.Job.Placeholders {
		token := "$" + strings.ToUpper(strings.ReplaceAll(ph, "-", "_"))
		switch binding.Kind {
		case BindingSecret:
			localCmd = strings.ReplaceAll(localCmd, token, "env:"+strings.ToUpper(strings.ReplaceAll(ph, "-", "_")))
		case BindingExpr:
			localCmd = strings.ReplaceAll(localCmd, token, "env:"+strings.ToUpper(strings.ReplaceAll(ph, "-", "_")))
		case BindingLiteral:
			localCmd = strings.ReplaceAll(localCmd, token, binding.Ref)
		}
	}
	return localCmd
}

// RenderDaggerMd produces a generic DAGGER.md local-usage doc using
// env:VAR placeholders for secrets.
func RenderDaggerMd(spec PipelineSpec) string {
	var b strings.Builder

	localCmd := renderLocalCommand(spec)

	b.WriteString("# dagger\n\n")
	b.WriteString("## Run ci on local\n\n")
	b.WriteString("It will run the following steps:\n")
	b.WriteString(fmt.Sprintf("  - Call `%s` function\n", spec.Job.Function))
	b.WriteString("\n\n```bash\n")
	b.WriteString("# Default local execution\n")
	b.WriteString(localCmd)
	b.WriteString(" export --path .\n")
	b.WriteString("```\n\n")
	b.WriteString("## Run ci without pushing the helm chart\n\n")
	b.WriteString("Same as above, but skip the helm chart push and the git commit/push.\n")
	b.WriteString("\n\n```bash\n")
	b.WriteString("# Dry-run execution\n")
	b.WriteString(localCmd)
	b.WriteString(" --dry-run true export --path .\n")
	b.WriteString("```\n")

	return b.String()
}
