package gadb

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/chzyer/readline"
)

// RunLocalShellMode enters the local shell REPL mode
// This mode provides a non-PTY shell experience with history and auto-completion
func RunLocalShellMode(ctx *Context) error {
	if !ctx.EnsureDevice() {
		return fmt.Errorf("no device selected")
	}

	device := ctx.CurrentDevice
	prompt := fmt.Sprintf("[%s] $ ", device.Serial)

	// Create readline instance for shell mode
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     getShellHistoryPath(),
		HistoryLimit:    100,
		AutoComplete:    getShellModeCompleter(),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	fmt.Println("")
	fmt.Printf("Entering shell mode for: %s\n", device.String())
	fmt.Println("Type 'exit', 'quit', or Ctrl+D to return to GADB")
	fmt.Println("For interactive commands (top, logcat), use: --pty")
	fmt.Println("")

	// Shell mode loop
	for {
		line, err := rl.Readline()
		if err != nil {
			// Handle Ctrl+D
			if err == readline.ErrInterrupt {
				if len(line) == 0 {
					fmt.Println("\nExiting shell mode...")
					break
				}
				continue
			}
			// EOF (Ctrl+D) or other error
			fmt.Println("\nExiting shell mode...")
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for exit commands
		if line == "exit" || line == "quit" || line == "q" {
			fmt.Println("Exiting shell mode...")
			break
		}

		// Check for --pty flag to switch to PTY mode
		if line == "--pty" || line == "-i" {
			fmt.Println("Switching to PTY mode...")
			ptyArgs := []string{"shell"}
			ExecWithPTY(device.Serial, ptyArgs)
			break
		}

		// Check if command ends with --pty for PTY execution of specific command
		if strings.HasSuffix(line, " --pty") || strings.HasSuffix(line, " -i") {
			// Extract the actual command
			var actualCmd string
			if strings.HasSuffix(line, " --pty") {
				actualCmd = strings.TrimSuffix(line, " --pty")
			} else {
				actualCmd = strings.TrimSuffix(line, " -i")
			}
			actualCmd = strings.TrimSpace(actualCmd)
			fmt.Printf("Running in PTY mode: %s\n", actualCmd)
			ptyArgs := []string{"shell", actualCmd}
			ExecWithPTY(device.Serial, ptyArgs)
			continue
		}

		// Execute the shell command
		if err := ExecSingleShellCommand(device, line); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	return nil
}

// ExecSingleShellCommand executes a single shell command on the device
// and streams the output in real-time
func ExecSingleShellCommand(device *Device, cmd string) error {
	// Use adb shell with the command
	args := []string{"-s", device.Serial, "shell", cmd}

	cmdExec := exec.Command("adb", args...)
	setupCommand(cmdExec)

	// Directly connect stdout and stderr for streaming output
	// This supports real-time commands like top, logcat, etc.
	cmdExec.Stdout = os.Stdout
	cmdExec.Stderr = os.Stderr
	cmdExec.Stdin = os.Stdin

	return cmdExec.Run()
}

// getShellHistoryPath returns the path to the shell history file
func getShellHistoryPath() string {
	tmpDir := os.TempDir()
	return tmpDir + "/gadb_shell_history"
}

// getShellModeCompleter returns a tab completer for shell mode commands
func getShellModeCompleter() *readline.PrefixCompleter {
	completers := make([]readline.PrefixCompleterInterface, 0)

	// Add all shell commands
	completers = append(completers, buildShellModeShellCompleter())

	// Add built-in commands
	completers = append(completers,
		readline.PcItem("exit"),
		readline.PcItem("quit"),
		readline.PcItem("q"),
		readline.PcItem("--pty"),
		readline.PcItem("-i"),
	)

	return readline.NewPrefixCompleter(completers...)
}

// buildShellModeShellCompleter builds the shell command completer for shell mode
// This is a simplified version that doesn't include the "shell" prefix
func buildShellModeShellCompleter() *readline.PrefixCompleter {
	subItems := make([]readline.PrefixCompleterInterface, 0)

	// pm with subcommands
	pmItems := make([]readline.PrefixCompleterInterface, len(pmCommands))
	for i, cmd := range pmCommands {
		pmItems[i] = readline.PcItem(cmd)
	}
	subItems = append(subItems, readline.PcItem("pm", pmItems...))

	// am with subcommands
	amItems := make([]readline.PrefixCompleterInterface, len(amCommands))
	for i, cmd := range amCommands {
		amItems[i] = readline.PcItem(cmd)
	}
	subItems = append(subItems, readline.PcItem("am", amItems...))

	// dumpsys with services
	dumpsysItems := make([]readline.PrefixCompleterInterface, len(dumpsysServices))
	for i, svc := range dumpsysServices {
		dumpsysItems[i] = readline.PcItem(svc)
	}
	subItems = append(subItems, readline.PcItem("dumpsys", dumpsysItems...))

	// Other common shell commands
	otherShellCmds := []string{
		"ps", "top", "getprop", "setprop",
		"wm", "input", "screencap", "screenrecord",
		"ls", "cd", "pwd", "cat", "grep", "rm", "mv", "cp", "mkdir", "mount", "umount",
		"netstat", "ping", "ifconfig", "ip", "route", "netcfg",
		"su", "id", "whoami", "date", "uptime", "sleep", "dmesg",
		"lsmod", "insmod", "rmmod", "kill", "killall",
		"chmod", "chown", "ln", "df", "du", "free", "uname",
		"settings", "service", "logcat",
	}
	for _, cmd := range otherShellCmds {
		subItems = append(subItems, readline.PcItem(cmd))
	}

	return readline.NewPrefixCompleter(subItems...)
}
