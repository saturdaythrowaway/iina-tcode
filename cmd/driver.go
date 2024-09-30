package main

import (
	"io"
	"os"
	"strings"
	"time"

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

func attemptReconnect() {
	dur := 1 * time.Second
	ticker := time.NewTicker(dur)

	for range ticker.C {
		err := connectToDevice()
		if err != nil {
			dur += time.Duration(float64(dur) * 0.2)

			if dur > 30*time.Second {
				dur = 30 * time.Second
			}

			log.Warn().Err(err).Msgf("failed to connect to device, retrying %d seconds", int(dur.Seconds()))

			ticker.Reset(dur)
		} else {
			log.Info().Msg("connected to device")

			return
		}
	}
}

func sendTCode(cmd string) error {
	cmd = strings.TrimSuffix(cmd, "\n")

	if cmd == "" {
		return nil
	}

	if port != nil {
		_, err := port.Write([]byte(cmd + "\n"))
		if err != nil {
			if strings.HasSuffix(err.Error(), "device not configured") {
				log.Warn().Err(err).Msg("device not configured, most likely disconnected")

				port = nil

				go attemptReconnect()

				return nil
			}

			return err
		}

		if os.Getenv("DEBUG") != "" {
			log.Debug().Str("tcode", cmd).Msg("sent")
		}
	}

	log.Trace().Str("tcode", cmd).Msg("tcode")

	return nil
}
