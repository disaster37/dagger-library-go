package helper

import (
	"fmt"
	"strings"
)

func ForgeCommand(cmd string) []string {
	return strings.Split(cmd, " ")
}

func ForgeScript(script string, params ...any) []string {
	return []string{
		"/bin/bash",
		"-c",
		fmt.Sprintf(script, params...),
	}
}
