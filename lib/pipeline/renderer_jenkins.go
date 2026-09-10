package pipeline

import (
	"fmt"
	"strings"

	"github.com/disaster37/dagger-library-go/lib/v2/ci"
)

// JenkinsRenderer produces Jenkins declarative pipeline Groovy.
type JenkinsRenderer struct{}

func (r *JenkinsRenderer) Supports(c ci.CI) bool {
	return c == ci.Jenkins
}

func (r *JenkinsRenderer) Render(spec PipelineSpec) (map[string]string, error) {
	g := &groovyBuilder{}

	registryCredential := ""
	gitTokenCredential := ""

	phMap := spec.Job.Placeholders
	if b, ok := phMap[PhRegistryUser]; ok && b.Kind == BindingSecret {
		registryCredential = b.Ref
	}
	if b, ok := phMap[PhGitToken]; ok && b.Kind == BindingSecret {
		gitTokenCredential = b.Ref
	}

	// Compute command args with Jenkins-specific resolver.
	// Branch/tag names are attacker-influenceable; use groovyString to prevent
	// GString injection and Groovy syntax breakage.
	escapedBranches := make([]string, len(spec.Branches))
	for i, br := range spec.Branches {
		escapedBranches[i] = groovyString(br)
	}
	escapedDefaultBranch := groovyEscapeDQ(spec.DefaultBranch)

	resolver := func(ph string, b Binding) string {
		switch ph {
		case PhVersion:
			return "${VERSION}"
		case PhBranch:
			return "${BRANCH_NAME}"
		case PhGitRepoURL:
			return "${GIT_URL}"
		case PhGitToken:
			return "env:GIT_CREDENTIAL_PSW"
		case PhRegistryUser:
			return "env:REGISTRY_CREDENTIAL_USR"
		case PhRegistryPass:
			return "env:REGISTRY_CREDENTIAL_PSW"
		default:
			switch b.Kind {
			case BindingSecret:
				envName := strings.ToUpper(strings.ReplaceAll(ph, "-", "_"))
				return "env:" + envName
			case BindingExpr:
				return b.Ref
			case BindingLiteral:
				return b.Ref
			}
			return ""
		}
	}

	args := buildCIArgs(spec, resolver)

	// Shell-quote ModuleRef, Job.Function, and literal arg values to prevent
	// command injection. ModuleRef uses single-quote shell quoting (not Groovy quotes).
	safeArgs := quoteArgsForShell(args)
	srcPrefix := ""
	if spec.SrcDir != "" {
		srcPrefix = fmt.Sprintf("--src %s ", shellQuote(spec.SrcDir))
	}
	fullCmd := fmt.Sprintf("dagger call -m %s %s%s %s", shellQuote(spec.ModuleRef), srcPrefix, shellQuote(spec.Job.Function), strings.Join(safeArgs, " "))
	if !spec.NoExport {
		fullCmd += " export --path ."
	}

	useKubernetes := spec.DaggerKubernetesToken != ""
	var kubeCmd string
	if useKubernetes {
		// Build command for daggerKubernetes step (single-quoted module ref, unquoted args)
		kubeSrcPrefix := ""
		if spec.SrcDir != "" {
			kubeSrcPrefix = fmt.Sprintf("--src %s ", spec.SrcDir)
		}
		kubeCmd = fmt.Sprintf("dagger call -m '%s' %s%s %s", spec.ModuleRef, kubeSrcPrefix, spec.Job.Function, strings.Join(args, " "))
		if !spec.NoExport {
			kubeCmd += " export --path ."
		}
	}

	// Build Groovy
	if useKubernetes {
		g.writeln("@Library('dagger-kubernetes') _")
		g.writeln("")
	}
	g.writeln("pipeline {")
	g.indent++

	// Environment block
	g.openBlock("environment")
	g.writeln(fmt.Sprintf("REGISTRY_CREDENTIAL = credentials(%s)", g.groovyString(registryCredential)))
	g.writeln(fmt.Sprintf("GIT_CREDENTIAL      = credentials(%s)", g.groovyString(gitTokenCredential)))
	g.writeln(`VERSION_TMP         = "${env.TAG_NAME == null ? "0.0.0-rc${BUILD_NUMBER}" : "${TAG_NAME.toLowerCase()}"}"`)
	g.writeln(`VERSION             = "${env.CHANGE_ID ==  null ? "${VERSION_TMP}" : "0.0.0-pr${CHANGE_ID}-${BUILD_NUMBER}"}"`)
	g.writeln(`BRANCH_NAME_TMP     = "${env.CHANGE_BRANCH == null ? "${GIT_BRANCH}" : "${CHANGE_BRANCH}"}"`)
	g.writeln(fmt.Sprintf(`BRANCH_NAME         = "${env.TAG_NAME == null ? "${BRANCH_NAME_TMP}" : "%s"}"`, escapedDefaultBranch))
	if useKubernetes {
		g.writeln(fmt.Sprintf("DAGGER_KUBERNETES_TOKEN = credentials(%s)", g.groovyString(spec.DaggerKubernetesToken)))
	}
	g.closeBlock()

	// Options block
	timeout := spec.TimeoutMinutes
	if timeout <= 0 {
		timeout = 10
	}
	g.openBlock("options")
	g.writeln(fmt.Sprintf("timeout time: %d, unit: 'MINUTES'", timeout))
	g.closeBlock()

	// Agent block
	g.openBlock("agent")
	g.openBlock("kubernetes")
	g.writeln("inheritFrom 'dagger'")
	g.writeln("defaultContainer 'dagger'")
	g.closeBlock()
	g.closeBlock()

	// Stages block
	g.openBlock("stages")

	stageName := spec.Job.Function
	if useKubernetes {
		stageName = "Dagger"
	}
	g.openBlock(fmt.Sprintf("stage(%s)", groovyString(stageName)))
	g.openBlock("when")
	g.writeln("beforeAgent true")

	// Build when condition matching old template.
	// Branch names are escaped with groovyString to prevent Groovy injection.
	if len(escapedBranches) == 1 {
		g.openBlock("anyOf")
		g.writeln(fmt.Sprintf("changeRequest target: %s", escapedBranches[0]))
		g.writeln(fmt.Sprintf("branch %s", escapedBranches[0]))
		g.writeln("tag '*'")
		g.closeBlock()
	} else {
		g.openBlock("anyOf")
		for _, branch := range escapedBranches {
			g.openBlock("anyOf")
			g.writeln(fmt.Sprintf("changeRequest target: %s", branch))
			g.writeln(fmt.Sprintf("branch %s", branch))
			g.closeBlock()
		}
		g.writeln("tag '*'")
		g.closeBlock()
	}

	g.closeBlock() // close when

	g.openBlock("steps")
	if useKubernetes {
		serverURL := spec.DaggerKubernetesURL
		if serverURL == "" {
			serverURL = "https://dagger.cloud"
		}
		g.writeln("daggerKubernetes(")
		g.indent++
		g.writeln(fmt.Sprintf("serverUrl: '%s',", serverURL))
		g.writeln("token: env.DAGGER_KUBERNETES_TOKEN,")
		g.writeln("provisionCli: true,")
		g.writeln("dynamicStages: true,")
		g.writeln(fmt.Sprintf(`command: "%s",`, kubeCmd))
		g.indent--
		g.writeln(")")
	} else {
		g.writeln(fmt.Sprintf(`sh "%s"`, fullCmd))
	}
	g.closeBlock() // close steps
	g.closeBlock() // close stage
	g.closeBlock() // close stages
	g.closeBlock() // close pipeline

	return map[string]string{
		"Jenkinsfile": g.b.String(),
	}, nil
}
