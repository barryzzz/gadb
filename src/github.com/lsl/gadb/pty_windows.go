//go:build windows

package gadb

import (
	"os"
	"os/exec"
)

// ExecWithPTY executes an adb command with PTY support for full interactivity
// On Windows, this falls back to regular execution since PTY is limited
func ExecWithPTY(deviceSerial string, args []string) error {
	// Build adb command with device serial
	adbArgs := append([]string{"-s", deviceSerial}, args...)
	cmd := exec.Command("adb", adbArgs...)

	// Windows has limited PTY support, use regular execution with stdin/stdout
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// setupCommand configures a command for proper execution on Windows
func setupCommand(cmd *exec.Cmd) {
	// No special setup needed on Windows
}
