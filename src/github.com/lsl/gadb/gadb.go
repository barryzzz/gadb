package gadb

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type devices struct {
	serial, usb, product, model, device string
}

//选择设备
func select_devices(devs []devices) []devices {
	count := len(devs)
	fmt.Println("Device list:")
	fmt.Println("0) All devices")
	for i := 0; i < count; i++ {
		fmt.Printf("%d) %s \t %s \n", i+1, devs[i].serial, devs[i].model)
	}
	fmt.Printf("q) Exit this operation\n")

	input := bufio.NewScanner(os.Stdin)
	fmt.Printf("select device:")
	input.Scan()
	var line = input.Text()
	switch line {
	case "0":
		return devs[0:]
	case "q":
		fmt.Println("exit")
		os.Exit(0)
		break
	default:
		var arrays []devices
		var c, err = strconv.Atoi(line)
		if err != nil || c < 0 || c > count {
			fmt.Printf("error input: %s, retry again\n", line)
			return select_devices(devs)
		}
		arrays = append(arrays, devs[c])
		return arrays
	}
	return nil

}

func read_args() []string {
	var args = os.Args[1:]
	if len(args) == 0 {
		fmt.Println("just use sadb as an alias for adb")
		os.Exit(0)
	}
	return args
}

//读取设备
func read_devices() []devices {

	var cmd = exec.Command("adb", "devices", "-l")
	var stdout, err = cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	cmd.Start()
	reader := bufio.NewReader(stdout)
	var arrays []devices
	var re = regexp.MustCompile("\\s+")
	for {
		line, e := reader.ReadString('\n')
		if e != nil || io.EOF == e {
			break
		}
		s := strings.Trim(line, "\n")
		if !strings.HasPrefix(s, "List of devices") && s != "" {
			var ss = re.Split(s, -1)
			var dv = devices{ss[0], ss[2], ss[3], ss[4], ss[5]}
			arrays = append(arrays, dv)
		}
	}
	return arrays
}

//执行shell 指令
func exec_adb_cmd_on_device(device string, args []string) {
	var temps []string
	temps = append(temps, "-s")
	temps = append(temps, device)
	temps = append(temps, args...)
	fmt.Println("adb ", temps)
	cmd := exec.Command("adb", temps...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
func Gadb() {
	var devices = read_devices()
	var args = read_args()
	var count = len(devices)

	if args[0] == "devices" {
		var s, _ = exec.Command("adb", "devices").Output()
		fmt.Println(strings.Trim(string(s), "\n"))
		os.Exit(0)
	}
	switch {
	case count > 1:
		dvs := select_devices(devices)
		for i := 0; i < len(dvs); i++ {
			exec_adb_cmd_on_device(dvs[i].serial, args)
		}

	case count == 1:
		exec_adb_cmd_on_device(devices[0].serial, args)
	default:
		fmt.Println("No device found")
		os.Exit(0)
	}

}
