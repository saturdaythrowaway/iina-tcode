package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var TPS = time.Duration(time.Second / 10)

func main() {
	port := flag.Int("port", 6800, "port to listen on")
	logfile := flag.String("logfile", "", "log file")
	flag.Parse()

	if logfile != nil && *logfile != "" {
		f, err := os.OpenFile(*logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}

		fmt.Println("logging to", *logfile)

		log.Logger = log.Output(f)
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	if os.Getenv("DEBUG") != "" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if len(flag.Args()) == 0 {
		fmt.Println("usage: tcode-player <script>")
		os.Exit(1)
	}

	command := flag.Args()[0]
	args := flag.Args()[1:]

	switch command {
	case "listen":
		err := connectToDevice()
		if err != nil {
			log.Warn().Err(err).Msg("failed to connect to device")
		}

		buf, _ := exec.Command("lsof", fmt.Sprintf("-i:%d", *port), "-sTCP:LISTEN", "-tPn").Output()
		pid := strings.TrimSuffix(string(buf), "\n")
		if string(pid) != "" {
			cmd := exec.Command("kill", "-9", pid)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err = cmd.Run()
			if err != nil {
				panic(err)
			}
		}

		err = listen(*port)
		if err != nil {
			panic(err)
		}
	case "render":
		if len(args) < 2 {
			fmt.Println("usage: tcode-player render <script> <output>")
			os.Exit(1)
		}

		fs, err := parse(args[0])
		if err != nil {
			panic(err)
		}

		err = renderFunscriptHeatmap(*fs, args[1])
		if err != nil {
			panic(err)
		}
	case "play":
		if len(args) == 0 {
			fmt.Println("usage: tcode-player play <dir>")
			os.Exit(1)
		}

		err := connectToDevice()
		if err != nil {
			log.Warn().Err(err).Msg("failed to connect to device")
		}

		err = play(args[0])
		if err != nil {
			panic(err)
		}
	case "tcode":
		if len(args) == 0 {
			fmt.Println("usage: tcode-player tcode <commands>")
			os.Exit(1)
		}

		err := connectToDevice()
		if err != nil {
			log.Warn().Err(err).Msg("failed to connect to device")
		}

		for _, cmd := range args {
			err := sendTCode(cmd)
			if err != nil {
				panic(err)
			}
		}

	default:
		fmt.Println("error: unknown command")
		os.Exit(1)
	}
}
