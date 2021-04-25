package main

import (
	"go-serial/goccnet"
)

func main() {
	d := goccnet.NewDevice(&goccnet.DeviceConfig{DeviceType: byte(0x03), Path: "/dev/ttyUSB0", Baud: 9600})
	d.Connect()
	// d.Poll()
	// time.Sleep(1 * time.Second)
	// d.Ack()
	// time.Sleep(1 * time.Second)
	d.GetBillTable()
	// time.Sleep(1 * time.Second)
	// d.Ack()
	// for {
	// 	switch d.Status {
	// 	case "Initialize":
	// 		d.Ack()
	// 		d.EnableBillTypes()
	// 		d.GetBillTable()
	// 		d.Poll()
	// 	default:
	// 		d.Ack()
	// 		d.Poll()
	// 	}
	// }
}
