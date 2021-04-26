package goccnet

import (
	"bytes"
	"io"
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
var subStates = map[byte]string{
	0x60: "due to Insertion",
	0x61: "due to Magnetic",
	0x62: "due to Ramained bill in head",
	0x63: "due to Multiplying",
	0x64: "due to Converting",
	0x65: "due to Identification",
	0x66: "due to Verification",
	0x67: "due to Optic",
	0x68: "due to Inhibit",
	0x69: "due to Capacity",
	0x6A: "due to Operation",
	0x6C: "due to Leght",

	0x50: "Stack Motor Failed",
	0x51: "Transport Motor Speed Failure",
	0x52: "Transport Motor Failure",
	0x53: "Aligning Motor Failure",
	0x54: "Cassette Status Failure",
	0x55: "Optic Cancl Failure",
	0x56: "Magnetic Canal Failure",
	0x5F: "Capacitance Canal Failure",
}

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
	// command *Command
	config     *DeviceConfig
	serialPort *serial.Port
	Status     string
	BillStack  chan int
	billTable  map[int]int
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
		config,
		nil,
		"",
		make(chan int),
		make(map[int]int),
	}
	return device
}

func (device *Device) Connect() {
	conf := &serial.Config{
		Name:        device.config.Path,
		Baud:        device.config.Baud,
		ReadTimeout: 300 * time.Millisecond,
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

func (device *Device) StartPoll() error {
	for {
		switch device.Status {
		case "Initialize":
			if err := device.Ack(); err != nil {
				return err
			}
			if err := device.EnableBillTypes(); err != nil {
				return err
			}
			if err := device.Poll(); err != nil {
				return err
			}
		default:
			if err := device.Ack(); err != nil {
				return err
			}
			if err := device.Poll(); err != nil {
				return err
			}
		}

	}
}

func (device *Device) Poll() error {
	var code byte = 0x33
	resp, err := device.Execute(code, nil)
	if err != nil {
		return err
	}
	frame := FrameBuffer{}

	frame.buildFrameFromResp(resp)
	if len(frame.DATA) > 0 {
		device.Status = states[frame.DATA[0]]
		if int(frame.LNG[0]) > 6 {
			device.Status += " " + subStates[frame.DATA[1]]
		}
	}
	// fmt.Printf("Response status:%v\n", device.Status)
	if len(device.billTable) == 0 {
		device.GetBillTable()
	}
	if frame.DATA[0] == 0x81 {
		device.BillStack <- device.billTable[int(frame.DATA[1])]
	}
	return err
}

func (frame *FrameBuffer) buildFrameFromResp(resp []byte) error {
	r := bytes.NewBuffer(resp)
	frame.SYNC = r.Next(1)
	frame.ADR = r.Next(1)
	frame.LNG = r.Next(1)
	lengh := int(frame.LNG[0])
	if lengh > 0 {
		frame.DATA = r.Next(lengh - 5)
		frame.CRC = r.Next(2)
	}
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
	// TODO реализовать интерфейс настройки приема вида купюр
	_, err := device.Execute(code, []byte{0xff, 0xff, 0xff, 0, 0, 0})
	return err
}

func (device *Device) RequestStatistic() error {
	var code byte = 0x60
	_, err := device.Execute(code, []byte{})
	return err
}

func (device *Device) GetBillTable() error {
	var code byte = 0x41
	resp, err := device.Execute(code, nil)
	if err != nil {
		return err
	}
	frame := FrameBuffer{}
	frame.buildFrameFromResp(resp)
	if len(frame.DATA) > 0 {
		for t := 0; t < 24; t++ {
			word := frame.DATA[t*5 : t*5+5]
			cur_nom := word[0]
			cur_pow := word[4]
			amount := int(cur_nom) * int(math.Pow10(int(cur_pow)))
			device.billTable[t] = amount
		}
	}
	// fmt.Printf("Bill table %v\n", BillTable)
	return nil
}

func (device *Device) Execute(code byte, data []byte) ([]byte, error) {
	cmd := bytes.NewBuffer([]byte{sync, device.config.DeviceType})
	cmd.Write([]byte{(byte)(len(data) + 6)})
	cmd.Write([]byte{code})
	cmd.Write(data)

	res := bytes.NewBuffer(cmd.Bytes())
	res.Write(getCRC16(cmd.Bytes()))
	// fmt.Printf("Request message buf: %x code: %v\n", res.Bytes(), commands[code])
	_, err := device.serialPort.Write(res.Bytes())
	if err != nil {
		log.Printf("Write error %v\n", err)
		return nil, err
	}

	buf := []byte{}
	tmpbuf := make([]byte, 256)
	if code != 0x00 { //ACK
		for {
			n, err := device.serialPort.Read(tmpbuf)

			buf = append(buf, tmpbuf[:n]...)
			if err != nil && err != io.EOF {
				return nil, err
			}
			if n == 0 {
				return buf, nil
			}
		}
	}
	return buf, nil
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
