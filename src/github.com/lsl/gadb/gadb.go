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
func selectDevices(devs []devices) []devices {
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
		if err != nil || c < 0 || c-1 >= count {
			fmt.Printf("error input: %s, retry again\n", line)
			return selectDevices(devs)
		}
		arrays = append(arrays, devs[c-1])
		return arrays
	}
	return nil

}

func readArgs() []string {
	var args = os.Args[1:]
	if len(args) == 0 {
		fmt.Println("just use gadb as an alias for adb")
		os.Exit(0)
	}
	if len(args) ==1{
		match,_:=regexp.MatchString("\\S*.apk",args[0])
		fmt.Println(match)
		if(match){
			args=append([]string{"install","-r"},args...)
		}
	}
	return args
}

//读取设备
func readDevices() []devices {

	var cmd = exec.Command("adb", "devices", "-l")
	var stdout, err = cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
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
			if len(ss) >= 4 {
				if ss[1] == "offline" {
					continue
				}
				var dv = devices{ss[0], ss[2], ss[3], ss[4], ss[5]}
				arrays = append(arrays, dv)
			}
		}
	}
	return arrays
}

//执行shell 指令
func execAdbCmdOnDevice(device string, args []string) {
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
	var devices = readDevices()
	var args = readArgs()
	var count = len(devices)

	if args[0] == "devices" {
		var s, _ = exec.Command("adb", "devices").Output()
		fmt.Println(strings.Trim(string(s), "\n"))
		os.Exit(0)
	}
	switch {
	case count > 1:
		dvs := selectDevices(devices)
		for i := 0; i < len(dvs); i++ {
			execAdbCmdOnDevice(dvs[i].serial, args)
		}

	case count == 1:
		execAdbCmdOnDevice(devices[0].serial, args)
	default:
		fmt.Println("No device found")
		os.Exit(0)
	}

}
