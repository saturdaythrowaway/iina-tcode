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

var TPS = time.Duration(time.Second / 60)

func main() {
	logWriter := os.Stderr

	port := flag.Int("port", 6800, "port to listen on")
	logfile := flag.String("logfile", "", "log file")
	loglevel := flag.String("loglevel", "info", "log level")
	logformat := flag.String("logformat", "text", "log format")
	flag.Parse()

	if os.Getenv("DEBUG") != "" {
		loglevel = &[]string{"debug"}[0]
	}

	switch *loglevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		fmt.Println("error: unknown log level")
	}

	if *logfile != "" {
		f, err := os.OpenFile(*logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
		if err != nil {
			panic(err)
		}

		logWriter = f
	}

	switch *logformat {
	case "json":
		log.Logger = log.Logger.With().Caller().Logger().Output(logWriter)
	case "text":
		log.Logger = log.Logger.With().Caller().Logger().Output(zerolog.ConsoleWriter{Out: logWriter})
	}

	log.Info().
		Str("arg0", os.Args[0]).
		Any("args", os.Args).
		Str("loglevel", zerolog.GlobalLevel().String()).
		Msg("starting tcode-player")

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
		if pid != "" {
			cmd := exec.Command("kill", "-9", pid)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err = cmd.Run()
			if err != nil {
				panic(err)
			}
		}

		time.Sleep(time.Millisecond)

		err = listen(*port)
		if err != nil {
			panic(err)
		}
	case "render":
		if len(args) < 2 {
			fmt.Println("usage: tcode-player render <script> <output>")
			os.Exit(1)
		}

		script, err := NewScript(args[0])
		if err != nil {
			panic(err)
		}

		err = renderFunscriptHeatmap(*script, args[1])
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
