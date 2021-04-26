package main

import (
	"fmt"
	"go-serial/goccnet"
	"log"
)

func main() {
	d := goccnet.NewDevice(&goccnet.DeviceConfig{DeviceType: byte(0x03), Path: "/dev/ttyUSB0", Baud: 9600})
	d.Connect()
	go stackInfo(d)
	if err := d.StartPoll(); err != nil {
		log.Fatal(err)
	}
}

func stackInfo(d *goccnet.Device) {
	for {
		select {
		case billStack := <-d.BillStack:
			fmt.Printf("Принято: %v\n", billStack)
		}
	}
}
