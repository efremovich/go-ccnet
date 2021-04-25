package goccnet

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
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

var commands = map[byte]string{
	0x30: "Reset",
	0x31: "GetStatus",
	0x32: "SetSecurity",
	0x33: "Poll",
	0x34: "EnableBillTypes",
	0x35: "Stack",
	0x36: "Return",
	0x37: "Identification",
	0x38: "Hold",
	0x41: "GetBillTable",
	0x60: "RequestStatistic",
	0x00: "Ack",
}

const sync = 0x02

type Device struct {
	isConnect bool
	busy      bool
	// command *Command
	config     *DeviceConfig
	serialPort *serial.Port
	Status     string
}

type FrameBuffer struct {
	SYNC []byte
	ADR  []byte
	LNG  []byte
	DATA []byte
	CRC  []byte
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
		"",
	}
	return device
}

func (device *Device) Connect() {
	conf := &serial.Config{
		Name:        device.config.Path,
		Baud:        device.config.Baud,
		ReadTimeout: 1000 * time.Millisecond,
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
	frame := FrameBuffer{}

	frame.buildFrameFromResp(resp)
	if len(frame.DATA) > 0 {
		device.Status = states[frame.DATA[0]]
	}
	fmt.Printf("Response s:%x a:%x l:%x d:%x crc:%x\n", frame.SYNC, frame.ADR, frame.LNG, frame.DATA, frame.CRC)
	return err
}

func (frame *FrameBuffer) buildFrameFromResp(resp []byte) error {
	fmt.Println("dddddddd", resp)
	frame.SYNC = resp[0:1]
	frame.ADR = resp[1:2]
	frame.LNG = resp[2:3]
	frame.DATA = resp[3 : int(frame.LNG[0])-5]
	frame.CRC = resp[int(frame.LNG[0])-2:]
	// r := bytes.NewBuffer(resp)
	// frame.SYNC = r.Next(1)
	// frame.ADR = r.Next(1)
	// frame.LNG = r.Next(1)
	// frame.CMD = r.Next(1)
	// lengh := int(frame.LNG[0])
	// if lengh > 0 {
	// 	frame.DATA = r.Next(lengh - 6)
	// 	frame.CRC = r.Next(2)
	// }
	return nil
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

func ByteToFloat64(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)

	return math.Float64frombits(bits)
}

func (device *Device) GetBillTable() error {
	var code byte = 0x41
	resp, err := device.Execute(code, nil)
	frame := FrameBuffer{}
	frame.buildFrameFromResp(resp)
	fmt.Println(frame.DATA)
	if len(frame.DATA) > 0 {
		for t := 0; t < 23; t++ {
			word := frame.DATA[t*5 : t*5+5]
			cur_nom := word[0]
			cur_pow := word[4]
			code := string(word[1:4])
			fmt.Printf("code %v nom %v pow %v data %v\n", code, cur_nom, cur_pow, word)
		}
	}

	fmt.Printf("Bill table %v\n", frame)
	return err
}

func (device *Device) Execute(code byte, data []byte) ([]byte, error) {
	cmd := bytes.NewBuffer([]byte{sync, device.config.DeviceType})
	cmd.Write([]byte{(byte)(len(data) + 6)})
	cmd.Write([]byte{code})
	cmd.Write(data)

	res := bytes.NewBuffer(cmd.Bytes())
	res.Write(getCRC16(cmd.Bytes()))
	fmt.Printf("Request message buf: %x code: %v\n", res.Bytes(), commands[code])
	n, err := device.serialPort.Write(res.Bytes())
	if err != nil {
		log.Printf("Write error %v\n", err)
		return nil, err
	}

	buf := []byte{}
	buf1 := make([]byte, 256)
	if code != 0x00 { //ACK
		i := 0
		l := 0
		for {
			n, err = device.serialPort.Read(buf1)
			if err != nil {
				return nil, err
			}
			buf = append(buf, buf1...)
			i += n
			fmt.Printf("%v %v\n", i, buf1[:i])
			if l == 0 {
				l = int(buf1[2:3][0])

			}
			if l == i {
				return buf, nil
			}
		}
	}
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
