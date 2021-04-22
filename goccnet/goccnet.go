package goccnet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/tarm/serial"
)

var states = map[byte]string{
	0x10: "Power UP",
	0x11: "Power Up with Bill in Validator",
	0x12: "Power Up with Bill in Stacker",
	0x13: "Initialize",
	0x14: "Idling",
	0x15: "Accepting",
	0x17: "Stacking",
	0x18: "Returning",
	0x19: "Unit Disabled",
	0x1A: "Holding",
	0x1B: "Device Busy",
	0x1C: "Rejecting",
	0x41: "Drop Cassette Full",
	0x42: "Drop Cassette out of position",
	0x43: "Validator Jammed",
	0x44: "Drop Cassette Jammed",
	0x45: "Cheated",
	0x46: "Pause",
	0x47: "Failed",
	0x80: "Escrow position",
	0x81: "Bill stacked",
	0x82: "Bill returned"}

var commands = map[string]byte{
	"Reset":            0x30,
	"GetStatus":        0x31,
	"SetSecurity":      0x32,
	"Poll":             0x33,
	"EnableBillTypes":  0x34,
	"Stack":            0x35,
	"Return":           0x36,
	"Identification":   0x37,
	"Hold":             0x38,
	"GetBillTable":     0x41,
	"RequestStatistic": 0x60,
}

const sync = 0x02

type Device struct {
	isConnect bool
	busy      bool
	// command *Command
	config     *DeviceConfig
	serialPort *serial.Port
	states     map[byte]string
}
type DeviceConfig struct {
	DeviceType byte
	Path       string
	Baud       int
}

func NewDevice(config *DeviceConfig) *Device {
	device := &Device{
		false,
		false,
		config,
		nil,
		states,
	}
	return device
}

func (device *Device) Connect() {
	conf := &serial.Config{
		Name:        device.config.Path,
		Baud:        device.config.Baud,
		ReadTimeout: 100 * time.Millisecond,
		Size:        0,
		Parity:      0,
		StopBits:    0,
	}
	s, err := serial.OpenPort(conf)
	if err != nil {
		log.Fatal(err)
	}
	device.serialPort = s
	err = device.Reset()
	if err != nil {
		log.Fatal(err)
	}
	device.isConnect = true
}
func (device *Device) Ack() error {
	var code byte = 0x00
	_, err := device.Execute(code, nil)
	return err
}

func (device *Device) Reset() error {
	var code byte = 0x30
	_, err := device.Execute(code, nil)
	return err
}

func (device *Device) Poll() error {
	var code byte = 0x33
	resp, err := device.Execute(code, nil)
	r := bytes.NewBuffer(resp)
	sync := r.Next(1)
	addr := r.Next(1)
	lng := r.Next(1)
	cmd := r.Next(1)
	lengh := int(lng[0])
	data := []byte{}
	crc := []byte{}
	if lengh > 0 {
		data = r.Next(int(lng[0]) - 6)
		crc = r.Next(2)
	}
	fmt.Printf("Poll %v %v %v %v %v %v\n", sync, addr, lng, states[cmd[0]], data, crc)
	return err
}

func (device *Device) Identification() error {
	var code byte = 0x37
	_, err := device.Execute(code, nil)
	return err
}

func (device *Device) SetSecurity() error {
	var code byte = 0x32
	_, err := device.Execute(code, []byte{0, 0, 0})
	return err
}

func (device *Device) GetStatus() error {
	var code byte = 0x31
	_, err := device.Execute(code, nil)
	return err
}

func (device *Device) EnableBillTypes() error {
	var code byte = 0x34
	_, err := device.Execute(code, []byte{0xff, 0xff, 0xff, 0, 0, 0})
	return err
}

func (device *Device) GetBillTable() error {
	var code byte = 0x41
	_, err := device.Execute(code, nil)
	return err
}

func (device *Device) Execute(code byte, data []byte) ([]byte, error) {
	cmd := bytes.NewBuffer([]byte{sync, device.config.DeviceType})
	code_arr := []byte{code}
	cmd.Write([]byte{(byte)(len(code_arr) + len(data) + 5)})
	cmd.Write([]byte{code})
	cmd.Write(data)

	res := bytes.NewBuffer(cmd.Bytes())
	res.Write(getCRC16(cmd.Bytes()))
	// fmt.Printf("Request message buf: %x code: %x\n", res.Bytes(), code)
	n, err := device.serialPort.Write(res.Bytes())
	if err != nil {
		log.Printf("Write error %v\n", err)
		return nil, err
	}

	buf := make([]byte, 256)
	n = 0
	for n == 0 {
		n, err = device.serialPort.Read(buf)
		if err != nil {
			return nil, err
		}
	}
	// fmt.Printf("Response message n: %v buf: %v code: %x\n", n, buf, code)
	// if n != 6 {
	// return nil, errors.New("received datagram with invalid size (must: 6, was: " + strconv.Itoa(n) + ")")
	// }
	return buf, nil
}

func timeOutPortScanner(ctx context.Context) error {
	time.Sleep(1 * time.Millisecond)
	return errors.New("time out 200 ms")
}

func getCRC16(data []byte) []byte {
	var POLYNOMIAL = 0x08408
	var buf bytes.Buffer
	var crc uint16
	for i := 0; i < len(data); i++ {
		crc = crc ^ uint16(data[i])
		for j := 0; j < 8; j++ {
			if (crc & 0x0001) == 1 {
				crc = crc >> 1
				crc ^= uint16(POLYNOMIAL)
			} else {
				crc = crc >> 1
			}
		}
	}
	buf.WriteByte(byte(crc))
	buf.WriteByte(byte(crc >> 8))
	crc_byte := buf.Bytes()
	return crc_byte

}
