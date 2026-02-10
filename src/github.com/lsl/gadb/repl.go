package gadb

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
)

// RunREPL starts the interactive REPL loop
func RunREPL(devices []Device) error {
	ctx := NewContext(devices)

	// If no devices, try to refresh
	if len(devices) == 0 {
		ctx.RefreshDevices()
		if len(ctx.AvailableDevices) == 0 {
			fmt.Println("No devices found. Exiting...")
			return nil
		}
	}

	// If only one device, auto-select it
	if len(ctx.AvailableDevices) == 1 {
		ctx.CurrentDevice = &ctx.AvailableDevices[0]
	}

	// If multiple devices and none selected, show selection
	if len(ctx.AvailableDevices) > 1 && ctx.CurrentDevice == nil {
		selected := selectDevices(ctx.AvailableDevices)
		if len(selected) == 0 {
			fmt.Println("No device selected. Exiting...")
			return nil
		}
		ctx.CurrentDevice = &selected[0]
	}

	// Show welcome message
	printWelcome(ctx)

	// Create readline instance
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          ctx.GetPrompt(),
		HistoryFile:     os.TempDir() + "/gadb_history",
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	// Main REPL loop
	for ctx.Running {
		// Update prompt in case device changed
		rl.SetPrompt(ctx.GetPrompt())

		line, err := rl.Readline()
		if err != nil {
			// Handle Ctrl+D
			if err == readline.ErrInterrupt {
				if len(line) == 0 {
					fmt.Println("\nExiting...")
					break
				}
				continue
			}
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			// Empty input - show current device status
			printDeviceStatus(ctx)
			continue
		}

		// Execute command
		if err := executeREPLInput(ctx, line); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	return nil
}

// executeREPLInput parses and executes REPL input
func executeREPLInput(ctx *Context, input string) error {
	ctx.AddToHistory(input)

	// Check for help command
	if input == "help" || input == "h" || input == "?" {
		printHelp()
		return nil
	}

	// Check for exit commands
	if input == "q" || input == "exit" || input == "quit" {
		fmt.Println("Exiting...")
		ctx.Stop(0)
		return nil
	}

	// Check if input is a pure number - switch device
	if idx, err := strconv.Atoi(input); err == nil {
		return switchDeviceByIndex(ctx, idx)
	}

	// Check if input starts with ':' - alternative device switch syntax
	if strings.HasPrefix(input, ":") {
		idxStr := strings.TrimPrefix(input, ":")
		if idx, err := strconv.Atoi(idxStr); err == nil {
			return switchDeviceByIndex(ctx, idx)
		}
	}

	// Pass through to adb
	if !ctx.EnsureDevice() {
		return fmt.Errorf("no device selected")
	}

	// Parse command for redirection and pipeline
	parsed := ParseCommand(input)
	return ExecWithRedirect(ctx.CurrentDevice, parsed)
}

// switchDeviceByIndex switches the current device by index
func switchDeviceByIndex(ctx *Context, idx int) error {
	ctx.RefreshDevices()

	if len(ctx.AvailableDevices) == 0 {
		fmt.Println("No devices found")
		return nil
	}

	// idx 0 means all devices (show list), idx 1+ means specific device
	if idx == 0 {
		printDeviceList(ctx)
		return nil
	}

	if idx < 1 || idx > len(ctx.AvailableDevices) {
		fmt.Printf("Invalid device index: %d\n", idx)
		printDeviceList(ctx)
		return nil
	}

	ctx.CurrentDevice = &ctx.AvailableDevices[idx-1]
	fmt.Printf("Switched to: %s\n", ctx.CurrentDevice.String())
	return nil
}

// printWelcome shows the welcome message
func printWelcome(ctx *Context) {
	fmt.Println("")
	fmt.Println("  GADB - Fast ADB Device Switcher")
	fmt.Println("")
	printDeviceStatus(ctx)
	fmt.Println("Commands:")
	fmt.Println("  <number>       - Switch to device (1, 2, 3...)")
	fmt.Println("  0              - Show device list")
	fmt.Println("  help           - Show detailed help")
	fmt.Println("  <adb cmd>      - Execute adb command on current device")
	fmt.Println("  q, exit        - Quit")
	fmt.Println("")
}

// printDeviceStatus shows the current device status
func printDeviceStatus(ctx *Context) {
	ctx.RefreshDevices()
	if ctx.CurrentDevice != nil {
		fmt.Printf("  Current: %s\n", ctx.CurrentDevice.String())
	} else {
		fmt.Println("  No device selected")
	}
	fmt.Printf("  Devices: %d connected\n", len(ctx.AvailableDevices))
	fmt.Println("")
}

// printDeviceList shows all available devices
func printDeviceList(ctx *Context) {
	fmt.Println("")
	for i, d := range ctx.AvailableDevices {
		prefix := "  "
		if ctx.CurrentDevice != nil && d.Serial == ctx.CurrentDevice.Serial {
			prefix = "* "
		}
		fmt.Printf("%s[%d] %s\n", prefix, i+1, d.String())
	}
	fmt.Println("")
}

// printHelp shows detailed help information
func printHelp() {
	fmt.Println("")
	fmt.Println("  GADB - Fast ADB Device Switcher")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("  gadb              - Start interactive REPL mode")
	fmt.Println("  gadb <command>    - Execute adb command on selected device")
	fmt.Println("  gadb devices      - List all connected devices")
	fmt.Println("")
	fmt.Println("REPL COMMANDS:")
	fmt.Println("  help, h, ?       - Show this help message")
	fmt.Println("  <number>         - Switch to device (1, 2, 3...)")
	fmt.Println("  0                - Show device list")
	fmt.Println("  Enter (empty)    - Show current device status")
	fmt.Println("  q, exit, quit    - Quit REPL")
	fmt.Println("")
	fmt.Println("ADB COMMANDS (passed through):")
	fmt.Println("  shell <cmd>      - Execute shell command")
	fmt.Println("  shell            - Enter interactive shell")
	fmt.Println("  logcat [args]    - View logcat output")
	fmt.Println("  install <apk>    - Install APK file")
	fmt.Println("  uninstall <pkg>  - Uninstall package")
	fmt.Println("  push <src> <dst> - Push file to device")
	fmt.Println("  pull <src> <dst> - Pull file from device")
	fmt.Println("  ...any adb cmd   - All other adb commands work too")
	fmt.Println("")
	fmt.Println("REDIRECTION & PIPELINE:")
	fmt.Println("  cmd > file       - Redirect output to file (overwrite)")
	fmt.Println("  cmd >> file      - Append output to file")
	fmt.Println("  cmd | grep x     - Pipe output to another command")
	fmt.Println("")
	fmt.Println("EXAMPLES:")
	fmt.Println("  shell ps                    - List processes")
	fmt.Println("  shell ps | grep com.android - Filter processes")
	fmt.Println("  logcat -d > log.txt         - Save logcat to file")
	fmt.Println("  install app.apk             - Install app")
	fmt.Println("")
}

// RunNormalMode executes gadb in normal (non-REPL) mode
func RunNormalMode(args []string) error {
	devices := readDevices()
	count := len(devices)

	// Special handling for 'devices' command
	if args[0] == "devices" {
		ListDevices()
		return nil
	}

	// Check if it's a single APK file argument
	if len(args) == 1 && strings.HasSuffix(args[0], ".apk") {
		args = append([]string{"install", "-r"}, args...)
	}

	// Parse the full command line for redirection/pipeline
	// Reconstruct the command line from args
	input := strings.Join(args, " ")
	parsed := ParseCommand(input)

	switch {
	case count > 1:
		// Multiple devices - need selection
		selected := selectDevices(devices)
		for _, d := range selected {
			if err := ExecWithRedirect(&d, parsed); err != nil {
				return err
			}
		}
	case count == 1:
		// Single device - execute directly
		return ExecWithRedirect(&devices[0], parsed)
	default:
		fmt.Println("No device found")
		return fmt.Errorf("no device found")
	}

	return nil
}
