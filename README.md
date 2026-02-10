# GADB

Fast ADB device switcher for multiple Android devices.

## Why?

[sadb](https://github.com/linroid/sadb) startup is too slow, so rewrite it in Go.

## Installation

```bash
go build -o gadb
```

Or download the release package for your platform:
- `gadb-win64-1.0.5.tar.gz` - Windows
- `gadb-linux64-1.0.5.tar.gz` - Linux
- `gadb-mac64-1.0.5.tar.gz` - macOS

Add the executable to your PATH:

**macOS / Linux:**
```bash
export PATH=${PATH}:/path/to/gadb/
```

**Windows:**
Add the executable path to System Environment Variables.

## Usage

### Normal Mode

Execute ADB commands directly:

```bash
# Single device - executes directly
gadb shell ps

# Multiple devices - select from menu
gadb install app.apk

# List all devices
gadb devices

# Output redirection
gadb shell ps > process.txt

# Pipeline
gadb shell ps | grep "com.example"

# APK auto-install
gadb app.apk
```

### REPL Mode (Interactive)

Start REPL by running without arguments:

```bash
gadb
```

**REPL Commands:**

| Command | Description |
|---------|-------------|
| `help`, `h`, `?` | Show detailed help |
| `1`, `2`, `3`... | Switch to device by number |
| `0` | Show device list |
| `Enter` (empty) | Show current device status |
| `q`, `exit` | Quit REPL |

**ADB Commands (passed through):**

| Command | Description |
|---------|-------------|
| `shell <cmd>` | Execute shell command |
| `shell` | Enter interactive shell |
| `logcat [args]` | View logcat output |
| `install <apk>` | Install APK file |
| `uninstall <pkg>` | Uninstall package |
| `push <src> <dst>` | Push file to device |
| `pull <src> <dst>` | Pull file from device |
| `...any adb cmd` | All other adb commands |

**Redirection & Pipeline:**

| Syntax | Description |
|--------|-------------|
| `cmd > file` | Redirect output to file (overwrite) |
| `cmd >> file` | Append output to file |
| `cmd | grep x` | Pipe output to another command |

### Examples

```bash
# Start REPL
gadb

# In REPL:
> 1                    # Switch to device 1
> shell ps             # List processes
> shell ps | grep com  # Filter processes
> logcat -d > log.txt  # Save logcat to file
> install app.apk      # Install app
> help                 # Show help
> q                    # Quit
```

## Features

- Fast device switching
- Interactive REPL mode
- Output redirection (`>`, `>>`)
- Pipeline support (`|`)
- PTY support for interactive shell/logcat
- Cross-platform (Windows, macOS, Linux)

## License

MIT
