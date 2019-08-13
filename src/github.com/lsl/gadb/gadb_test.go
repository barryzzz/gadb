package gadb

import (
	"fmt"
	"testing"
	"time"
)

func Test_read_devices(t *testing.T) {
	var devices = readDevices()
	if devices != nil {

	}
}

func Test_read_devices_time(t *testing.T) {

	start := time.Now()
	read_devices()
	end := time.Now()
	fmt.Printf("time %s", end.Sub(start))
}
