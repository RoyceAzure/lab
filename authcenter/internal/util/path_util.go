package util

import (
	"os/exec"
	"path/filepath"
	"strings"
)

func GetProjectRoot(moduleName string) string {
	// 執行 go list，但是加上額外的過濾條件
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", moduleName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func WindwosPathToURL(winPath string) string {
	return filepath.ToSlash(winPath)
}
