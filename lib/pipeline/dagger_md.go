package pipeline

import (
	"fmt"
	"strings"
)

var credentialPlaceholders = map[string]bool{
	PhRegistryUser: true,
	PhRegistryPass: true,
	PhGitToken:     true,
}

func filterMinimalArgs(args []string) []string {
	var filtered []string
	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if arg == "--ci" {
			if i+1 < len(args) {
				skipNext = true
			}
			continue
		}
		if strings.HasPrefix(arg, "--ci=") {
			continue
		}
		if i+1 < len(args) {
			nextVal := args[i+1]
			for ph := range credentialPlaceholders {
				if strings.Contains(nextVal, "{{"+ph+"}}") {
					skipNext = true
					break
				}
			}
			if skipNext {
				continue
			}
		}
		filtered = append(filtered, arg)
	}
	return filtered
}

func buildLocalCmdWithArgs(spec PipelineSpec, customArgs []string) string {
	var parts []string
	parts = append(parts, "dagger", "call", "-m", shellQuote(spec.ModuleRef))
	if spec.SrcDir != "" {
		parts = append(parts, "--src", shellQuote(spec.SrcDir))
	}
	parts = append(parts, shellQuote(spec.Job.Function))

	for _, arg := range customArgs {
		processed := arg
		for placeholder, binding := range spec.Job.Placeholders {
			token := "{{" + placeholder + "}}"
			envName := strings.ToUpper(strings.ReplaceAll(placeholder, "-", "_"))
			if strings.Contains(processed, token) {
				switch binding.Kind {
				case BindingLiteral:
					processed = strings.ReplaceAll(processed, token, binding.Ref)
				default:
					processed = strings.ReplaceAll(processed, token, "env:"+envName)
				}
			}
		}
		parts = append(parts, processed)
	}

	return strings.Join(parts, " ")
}

func exportSuffix(spec PipelineSpec) string {
	if spec.NoExport {
		return ""
	}
	return " export --path ."
}

// RenderDaggerMd produces a module-specific DAGGER.md with 3 usage levels.
func RenderDaggerMd(spec PipelineSpec) string {
	var b strings.Builder

	minArgs := filterMinimalArgs(spec.Job.Args)
	level1 := buildLocalCmdWithArgs(spec, minArgs) + exportSuffix(spec)
	level2 := buildLocalCmdWithArgs(spec, append(minArgs,
		"--registry-username", "env:SU_USERNAME",
		"--registry-password", "env:SU_PASSWORD",
	)) + exportSuffix(spec)
	level3 := buildLocalCmdWithArgs(spec, append(minArgs,
		"--registry-username", "env:SU_USERNAME",
		"--registry-password", "env:SU_PASSWORD",
		"--ci", string(spec.CI),
	)) + exportSuffix(spec)

	desc := spec.Description
	if desc == "" {
		desc = "CI pipeline"
	}

	b.WriteString("# dagger\n\n")
	b.WriteString("## " + desc + "\n\n")
	b.WriteString("It will run the following steps:\n")
	b.WriteString(fmt.Sprintf("  - Call `%s` function\n", spec.Job.Function))
	b.WriteString("\n### 1. Minimal execution (no credentials)\n\n")
	b.WriteString("```bash\n")
	b.WriteString(level1)
	b.WriteString("\n```\n")
	b.WriteString("\n### 2. With credentials\n\n")
	b.WriteString("```bash\n")
	b.WriteString(level2)
	b.WriteString("\n```\n")
	b.WriteString("\n### 3. Full CI execution\n\n")
	b.WriteString("```bash\n")
	b.WriteString(level3)
	b.WriteString("\n```\n")

	return b.String()
}
