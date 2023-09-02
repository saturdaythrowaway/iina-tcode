package main

import (
	"io"
	"strings"

	"github.com/jacobsa/go-serial/serial"
	"github.com/rs/zerolog/log"
)

var port io.ReadWriteCloser

func connectToDevice() error {
	p, err := serial.Open(serial.OpenOptions{
		PortName:        "/dev/cu.usbserial-0001",
		BaudRate:        115200,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
		ParityMode:      serial.PARITY_NONE,
	})
	if err != nil {
		return err
	}

	port = p

	return nil
}

func sendTCode(cmd string) error {
	cmd = strings.TrimSuffix(cmd, "\n")

	if cmd == "" {
		return nil
	}

	if port != nil {
		_, err := port.Write([]byte(cmd + "\n"))
		if err != nil {
			return err
		}
	}

	log.Trace().Str("tcode", cmd).Msg("tcode")

	return nil
}
