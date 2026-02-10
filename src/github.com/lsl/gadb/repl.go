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

	// Create readline instance with completer
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          ctx.GetPrompt(),
		HistoryFile:     os.TempDir() + "/gadb_history",
		HistoryLimit:    100,
		AutoComplete:    getCompleter(),
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

// Common adb commands for auto-completion
var adbCommands = []string{
	"shell", "install", "uninstall", "push", "pull",
	"logcat", "bugreport", "devices", "connect", "disconnect",
	"forward", "reverse", "port-forward", "jdwp",
	"backup", "restore", "sync",
	"kill-server", "start-server", "version",
	"root", "unroot", "remount", "reboot",
	"tcpip", "usb", "wait-for-device",
}

// Shell subcommands for auto-completion
var shellCommands = []string{
	"ps", "top", "getprop", "setprop", "dumpsys", "pm", "am",
	"wm", "input", "screencap", "screenrecord", "ls", "cd", "pwd",
	"cat", "grep", "rm", "mv", "cp", "mkdir", "mount", "umount",
	"netstat", "ping", "ifconfig", "ip", "route", "netcfg",
	"su", "id", "whoami", "date", "uptime", "sleep", "dmesg",
	"lsmod", "insmod", "rmmod", "kill", "killall",
	"chmod", "chown", "ln", "df", "du", "free", "uname",
}

// pm (package manager) subcommands
var pmCommands = []string{
	"list", "list packages", "list packages -3", "list packages -s",
	"path", "install", "uninstall",
	"clear", "enable", "disable", "disable-user",
	"hide", "unhide",
	"grant", "revoke",
	"set-install-location", "get-install-location",
	"trim-caches", "create-user", "remove-user",
	"get-max-users", "dump",
}

// am (activity manager) subcommands
var amCommands = []string{
	"start", "startservice", "stopservice",
	"broadcast", "force-stop",
	"kill", "kill-all",
	"start-activity", "start-activity-as-user",
	"startservice", "startserviceasuser",
	"stopservice",
	"broadcast", "broadcast-as-user",
	"instrument",
	"dumpheap", "set-debug-app", "clear-debug-app",
	"monitor", "profile", "dump",
	"to-uri", "to-intent-uri",
}

// dumpsys services
var dumpsysServices = []string{
	"activity", "window", "package", "power",
	"battery", "cpuinfo", "meminfo", "procstats",
	"wifi", "network_management", "connectivity",
	"telephony", "phone", "bluetooth", "location",
	"audio", "media", "camera", "input",
	"alarm", "notification", "jobqueue",
}

// logcat options
var logcatOptions = []string{
	"-v", "-v time", "-v threadtime", "-v brief", "-v process", "-v tag", "-v thread", "-v raw", "-v long", "-v descriptive",
	"-s", "-f", "-r", "-n", "-t", "-d", "-g", "-G", "-c", "-b", "-B",
	"*:V", "*:D", "*:I", "*:W", "*:E",
	"*:S", "AndroidRuntime:E", "System.err:W",
}

// install options
var installOptions = []string{
	"-l", "-r", "-R", "-i", "-t", "-s", "-d", "-g", "--fastdeploy",
}

// uninstall options
var uninstallOptions = []string{
	"-k",
}

// push/pull options
var fileTransferOptions = []string{
	"-p", "-a", "-z", "-Z",
}

// getCompleter returns a tab completer for commands
func getCompleter() *readline.PrefixCompleter {
	completers := make([]readline.PrefixCompleterInterface, 0)

	// shell with nested subcommands
	completers = append(completers, buildShellCompleter())
	completers = append(completers, buildLogcatCompleter())
	completers = append(completers, buildInstallCompleter())
	completers = append(completers, buildUninstallCompleter())
	completers = append(completers, buildPushCompleter())
	completers = append(completers, buildPullCompleter())

	// Other adb commands
	for _, cmd := range adbCommands {
		if cmd == "shell" || cmd == "logcat" || cmd == "install" || cmd == "uninstall" || cmd == "push" || cmd == "pull" {
			continue
		}
		completers = append(completers, readline.PcItem(cmd))
	}

	// Add built-in commands
	completers = append(completers,
		readline.PcItem("help"),
		readline.PcItem("h"),
		readline.PcItem("?"),
		readline.PcItem("exit"),
		readline.PcItem("q"),
		readline.PcItem("quit"),
	)

	return readline.NewPrefixCompleter(completers...)
}

// buildShellCompleter builds shell command completer with nested subcommands
func buildShellCompleter() *readline.PrefixCompleter {
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

	// other shell commands
	otherShellCmds := []string{
		"ps", "top", "getprop", "setprop",
		"wm", "input", "screencap", "screenrecord",
		"ls", "cd", "pwd", "cat", "grep", "rm", "mv", "cp", "mkdir", "mount", "umount",
		"netstat", "ping", "ifconfig", "ip", "route", "netcfg",
		"su", "id", "whoami", "date", "uptime", "sleep", "dmesg",
		"lsmod", "insmod", "rmmod", "kill", "killall",
		"chmod", "chown", "ln", "df", "du", "free", "uname",
	}
	for _, cmd := range otherShellCmds {
		subItems = append(subItems, readline.PcItem(cmd))
	}

	return readline.PcItem("shell", subItems...)
}

// buildLogcatCompleter builds logcat command completer
func buildLogcatCompleter() *readline.PrefixCompleter {
	items := make([]readline.PrefixCompleterInterface, len(logcatOptions))
	for i, opt := range logcatOptions {
		items[i] = readline.PcItem(opt)
	}
	return readline.PcItem("logcat", items...)
}

// buildInstallCompleter builds install command completer
func buildInstallCompleter() *readline.PrefixCompleter {
	items := make([]readline.PrefixCompleterInterface, len(installOptions))
	for i, opt := range installOptions {
		items[i] = readline.PcItem(opt)
	}
	return readline.PcItem("install", items...)
}

// buildUninstallCompleter builds uninstall command completer
func buildUninstallCompleter() *readline.PrefixCompleter {
	items := make([]readline.PrefixCompleterInterface, len(uninstallOptions))
	for i, opt := range uninstallOptions {
		items[i] = readline.PcItem(opt)
	}
	return readline.PcItem("uninstall", items...)
}

// buildPushCompleter builds push command completer
func buildPushCompleter() *readline.PrefixCompleter {
	items := make([]readline.PrefixCompleterInterface, len(fileTransferOptions))
	for i, opt := range fileTransferOptions {
		items[i] = readline.PcItem(opt)
	}
	return readline.PcItem("push", items...)
}

// buildPullCompleter builds pull command completer
func buildPullCompleter() *readline.PrefixCompleter {
	items := make([]readline.PrefixCompleterInterface, len(fileTransferOptions))
	for i, opt := range fileTransferOptions {
		items[i] = readline.PcItem(opt)
	}
	return readline.PcItem("pull", items...)
}
