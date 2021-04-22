package main

import (
	"go-serial/goccnet"
)

func main() {
	d := goccnet.NewDevice(&goccnet.DeviceConfig{DeviceType: byte(0x03), Path: "/dev/ttyUSB0", Baud: 9600})
	d.Connect()
	d.Ack()
	d.EnableBillTypes()
	d.GetBillTable()
	d.Poll()
	for {
		d.Ack()
		d.Poll()
	}
}
