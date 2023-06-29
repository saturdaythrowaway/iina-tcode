package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type MethodCall struct {
	XMLName    xml.Name `xml:"methodCall"`
	Text       string   `xml:",chardata"`
	MethodName string   `xml:"methodName"`
	Params     struct {
		Text  string `xml:",chardata"`
		Param []struct {
			Text  string `xml:",chardata"`
			Value struct {
				Text   string `xml:",chardata"`
				String string `xml:"string"`
				Double string `xml:"double"`
			} `xml:"value"`
		} `xml:"param"`
	} `xml:"params"`
}

func (m MethodCall) GetParam(key string) string {
	k := strings.ToLower(key)
	for i, param := range m.Params.Param {
		if strings.ToLower(param.Value.String) == k {
			return m.Params.Param[i+1].Value.String
		}
	}

	return ""
}

func respond(w http.ResponseWriter, msg string) {
	_, err := w.Write([]byte(msg))
	if err != nil {
		panic(err)
	}
}

func listen(port int) error {
	var loadedScripts *Scripts
	var tcode *TCode

	var params = Params{}

	http.HandleFunc("/xmlrpc", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			panic(err)
		}

		var call MethodCall
		err = xml.Unmarshal(body, &call)
		if err != nil {
			panic(err)
		}

		switch call.MethodName {
		case "close": // no args
			tcode.Close()
		case "seek": // no args
			seek := call.GetParam("seek")
			if seek != "" {
				ts, err := time.ParseDuration(seek)
				if err != nil {
					log.Error().Err(err).Str("seek", seek).Msg("failed to parse seek")

					return
				}

				tcode.Seek(ts)
			}
		case "version": // no args
			respond(w, "1.0")
		case "load": // L#,R#,V#,script
			dir := call.GetParam("dir")
			if dir == "" {
				dir = call.GetParam("folder")
			}

			if dir == "" {
				respond(w, "no folder or dir param")
				return
			}

			loadedScripts = &Scripts{}
			err := loadedScripts.Load(dir)
			if err != nil {
				log.Error().Err(err).Msg("failed to load scripts")
				respond(w, err.Error())
			} else {
				respond(w, "ok")
			}

			if tcode != nil {
				tcode.Close()
			}

			tcode, err = loadedScripts.TCode(params)
			if err != nil {
				panic(err)
			}

			go func() {
				defer func() {
					_ = recover() // ignore panic
				}()

				for msg := range tcode.Tick() {
					err := sendTCode(msg)
					if err != nil {
						panic(err)
					}
				}
			}()
		case "set":

		case "render": // output
			if loadedScripts == nil {
				_, err = w.Write([]byte("no loaded script"))
				if err != nil {
					panic(err)
				}

				return
			}

			if loadedScripts.Stroke.Default == nil {
				return
			}

			err = renderFunscriptHeatmap(*loadedScripts.Stroke.Default, call.GetParam("output"))
			if err != nil {
				panic(err)
			}

			respond(w, "ok")
		case "pause": // seek
			if tcode == nil {
				respond(w, "file not loaded")
			} else {
				respond(w, "ok")

				tcode.Pause()

				seek := call.GetParam("seek")
				if seek != "" {
					ts, err := time.ParseDuration(seek)
					if err != nil {
						log.Error().Err(err).Str("seek", seek).Msg("failed to parse seek")

						return
					}

					tcode.Seek(ts)
				}
			}
		case "play": // seek
			if tcode == nil {
				respond(w, "file not loaded")
			} else {
				respond(w, "ok")

				seek := call.GetParam("seek")
				if seek != "" {
					ts, err := time.ParseDuration(seek)
					if err != nil {
						log.Error().Err(err).Str("seek", seek).Msg("failed to parse seek")

						return
					}

					tcode.Seek(ts)
				}

				tcode.Play()
			}
		default:
			log.Debug().Msgf("%s %s %s", r.RemoteAddr, call.MethodName, call.Params)
		}
	})

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		panic(err)

	}

	os.Exit(1)

	return nil
}
