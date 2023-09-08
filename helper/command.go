package helper

import "strings"

func ForgeCommand(cmd string) []string {
	return strings.Split(cmd, " ")
}

func ForgeScript(script string) []string {
	return []string{
		"/bin/bash",
		"-c",
		script,
	}
}
