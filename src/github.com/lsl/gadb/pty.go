package gadb

import (
	"fmt"
	"os"
	"os/exec"
)

// IsInteractiveCommand checks if the given command requires PTY support
func IsInteractiveCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	cmd := args[0]
	// These commands require full PTY support
	interactiveCmds := map[string]bool{
		"shell":  true,
		"sh":     true,
		"logcat": true, // logcat needs PTY for proper Ctrl+C handling
	}
	return interactiveCmds[cmd]
}

// ExecCommand executes a command on the specified device
// Automatically uses PTY for interactive commands
func ExecCommand(device *Device, args []string) error {
	if device == nil {
		return fmt.Errorf("no device specified")
	}

	if IsInteractiveCommand(args) {
		// Use PTY for interactive commands
		return ExecWithPTY(device.Serial, args)
	}

	// Regular command execution
	adbArgs := append([]string{"-s", device.Serial}, args...)
	cmd := exec.Command("adb", adbArgs...)

	// Platform-specific command setup (e.g., process group on Windows)
	setupCommand(cmd)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("adb %s\n", adbArgs)
	return cmd.Run()
}

// ExecCommandOnAll executes a command on all available devices
func ExecCommandOnAll(devices []Device, args []string) error {
	for _, d := range devices {
		if err := ExecCommand(&d, args); err != nil {
			return err
		}
	}
	return nil
}
