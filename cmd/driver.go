package main

import (
	"io"

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
	if cmd == "" {
		return nil
	}

	if cmd[len(cmd)-1] != '\n' {
		cmd += "\n"
	}

	if port != nil {
		_, err := port.Write([]byte(cmd))
		if err != nil {
			return err
		}
	}

	log.Debug().Str("tcode", cmd).Msg("sent tcode")

	return nil
}
