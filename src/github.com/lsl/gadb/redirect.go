package gadb

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RedirectMode represents the type of output redirection
type RedirectMode int

const (
	RedirectNone  RedirectMode = iota
	RedirectOverwrite           // >
	RedirectAppend              // >>
)

// ParsedCommand represents a command with potential redirection or pipeline
type ParsedCommand struct {
	Args      []string      // Command arguments (before redirection/pipe)
	Redirect  RedirectMode   // Output redirection mode
	RedirectFile string      // Output file path (for redirection)
	PipeCmd   []string      // Piped command (for pipeline)
}

// ParseCommand parses a command line string for redirection and pipeline operators
func ParseCommand(input string) *ParsedCommand {
	parsed := &ParsedCommand{Redirect: RedirectNone}

	// Check for pipe first (highest precedence)
	if pipeIdx := indexOfPipe(input); pipeIdx != -1 {
		leftPart := strings.TrimSpace(input[:pipeIdx])
		rightPart := strings.TrimSpace(input[pipeIdx+1:])
		parsed.Args = strings.Fields(leftPart)
		parsed.PipeCmd = strings.Fields(rightPart)
		return parsed
	}

	// Check for append redirection >>
	if appendIdx := indexOfAppend(input); appendIdx != -1 {
		leftPart := strings.TrimSpace(input[:appendIdx])
		rightPart := strings.TrimSpace(input[appendIdx+2:])
		parsed.Args = strings.Fields(leftPart)
		parsed.Redirect = RedirectAppend
		parsed.RedirectFile = rightPart
		return parsed
	}

	// Check for overwrite redirection >
	if redirectIdx := indexOfRedirect(input); redirectIdx != -1 {
		leftPart := strings.TrimSpace(input[:redirectIdx])
		rightPart := strings.TrimSpace(input[redirectIdx+1:])
		parsed.Args = strings.Fields(leftPart)
		parsed.Redirect = RedirectOverwrite
		parsed.RedirectFile = rightPart
		return parsed
	}

	// No special operators, just parse as regular command
	parsed.Args = strings.Fields(input)
	return parsed
}

// indexOfRedirect finds the index of > that is not part of >>
// Returns -1 if not found
func indexOfRedirect(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '>' {
			// Check if this is part of >>
			if i+1 < len(s) && s[i+1] == '>' {
				continue
			}
			// Make sure it's not inside quotes
			if !isInQuotes(s, i) {
				return i
			}
		}
	}
	return -1
}

// indexOfAppend finds the index of >>
// Returns -1 if not found
func indexOfAppend(s string) int {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '>' && s[i+1] == '>' {
			if !isInQuotes(s, i) {
				return i
			}
		}
	}
	return -1
}

// indexOfPipe finds the index of |
// Returns -1 if not found
func indexOfPipe(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '|' {
			if !isInQuotes(s, i) {
				return i
			}
		}
	}
	return -1
}

// isInQuotes checks if the index is inside quotes
func isInQuotes(s string, idx int) bool {
	inSingle := false
	inDouble := false
	for i := 0; i < idx; i++ {
		if s[i] == '\'' && !inDouble {
			inSingle = !inSingle
		}
		if s[i] == '"' && !inSingle {
			inDouble = !inDouble
		}
	}
	return inSingle || inDouble
}

// ExecWithRedirect executes a command with redirection support
func ExecWithRedirect(device *Device, parsed *ParsedCommand) error {
	if device == nil {
		return fmt.Errorf("no device specified")
	}

	// Check for pipeline
	if len(parsed.PipeCmd) > 0 {
		return execPipeline(device, parsed)
	}

	// Check for redirection
	if parsed.Redirect != RedirectNone {
		return execWithFileRedirect(device, parsed)
	}

	// No redirection or pipeline, use normal execution
	return ExecCommand(device, parsed.Args)
}

// execWithFileRedirect executes command with output redirected to a file
func execWithFileRedirect(device *Device, parsed *ParsedCommand) error {
	// Open output file
	var file *os.File
	var err error

	if parsed.Redirect == RedirectAppend {
		file, err = os.OpenFile(parsed.RedirectFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(parsed.RedirectFile)
	}

	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer file.Close()

	// For PTY commands, we can't easily redirect, so use non-PTY mode
	if IsInteractiveCommand(parsed.Args) {
		// Build adb command
		adbArgs := append([]string{"-s", device.Serial}, parsed.Args...)
		cmd := exec.Command("adb", adbArgs...)
		setupCommand(cmd)

		cmd.Stdin = os.Stdin
		cmd.Stdout = file
		cmd.Stderr = file

		return cmd.Run()
	}

	// Use non-PTY execution with redirection
	adbArgs := append([]string{"-s", device.Serial}, parsed.Args...)
	cmd := exec.Command("adb", adbArgs...)
	setupCommand(cmd)

	cmd.Stdin = os.Stdin
	cmd.Stdout = file
	cmd.Stderr = os.Stderr // Keep stderr on console

	return cmd.Run()
}

// execPipeline executes a command pipeline: adb cmd | other_cmd
func execPipeline(device *Device, parsed *ParsedCommand) error {
	// For PTY commands, we need to capture output differently
	if IsInteractiveCommand(parsed.Args) {
		return execPipelinePTY(device, parsed)
	}

	// First command: adb command
	adbArgs := append([]string{"-s", device.Serial}, parsed.Args...)
	cmd1 := exec.Command("adb", adbArgs...)
	setupCommand(cmd1)

	// Create pipe
	stdout1, err := cmd1.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}

	// Second command: the piped command
	cmd2 := exec.Command(parsed.PipeCmd[0], parsed.PipeCmd[1:]...)
	cmd2.Stdin = stdout1
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr

	// Start first command
	if err := cmd1.Start(); err != nil {
		return fmt.Errorf("failed to start adb command: %w", err)
	}

	// Run second command
	if err := cmd2.Run(); err != nil {
		return fmt.Errorf("failed to run piped command: %w", err)
	}

	// Wait for first command to complete
	if err := cmd1.Wait(); err != nil {
		return fmt.Errorf("adb command failed: %w", err)
	}

	return nil
}

// execPipelinePTY executes a pipeline with PTY commands (like shell)
func execPipelinePTY(device *Device, parsed *ParsedCommand) error {
	// For PTY commands, we need to capture output in memory
	adbArgs := append([]string{"-s", device.Serial}, parsed.Args...)
	cmd := exec.Command("adb", adbArgs...)
	setupCommand(cmd)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		// Don't error on exit, just continue with pipeline
	}

	// Pipe output to second command
	cmd2 := exec.Command(parsed.PipeCmd[0], parsed.PipeCmd[1:]...)
	cmd2.Stdin = &output
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	cmd2.Stdin = bytes.NewReader(output.Bytes())

	return cmd2.Run()
}
