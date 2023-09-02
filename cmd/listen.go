package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
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

func respond(w http.ResponseWriter, status int, msg string) {
	template := `<?xml version="1.0"?>
	<methodResponse>
	   <params>
		  <param>
			 <value><string>%s</string></value>
		  </param>
	   </params>
	</methodResponse>`

	w.WriteHeader(status)

	_, err := w.Write([]byte(fmt.Sprintf(template, msg)))
	if err != nil {
		panic(err)
	}
}

func listen(port int) {
	var (
		loadedScripts *Scripts
		tcode         *TCode
	)

	closeChan := make(chan bool)

	// todo: add jsonrpc & grpc (?)

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
			closeChan <- true
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
			respond(w, http.StatusOK, "1.0")
		case "load": // L#,R#,V#,script
			filename := call.GetParam("filename")
			dir := call.GetParam("folder")

			path := filename
			if path == "" {
				path = dir
			}

			if path == "" {
				log.Error().Msg("no filename or folder param")
				respond(w, http.StatusInternalServerError, "no folder or dir param")

				return
			}

			log.Debug().Str("filename", filename).Str("dir", dir).Msg("load")

			loadedScripts = &Scripts{
				preferedModifier: ScriptModSoft,
			}
			err := loadedScripts.Load(path)
			if err != nil {
				log.Error().Err(err).Msg("failed to load scripts")
				respond(w, http.StatusInternalServerError, err.Error())

				return
			}

			log.Debug().Strs("scripts", loadedScripts.Loaded()).Msg("loaded scripts")

			respond(w, http.StatusOK, fmt.Sprintf("loaded %v", loadedScripts.Loaded()))

			if tcode != nil {
				tcode.Reset()
			}

			tcode, err = loadedScripts.TCode()
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
			l := log.Debug()
			change := false

			min := call.GetParam("min")
			if min != "" {
				f, err := strconv.ParseFloat(min, 64)
				if err != nil {
					log.Error().Err(err).Str("min", min).Msg("failed to parse min")
				} else {
					if f != params.Min {
						l.Float64("min", f)
						change = true
					}

					params.Min = f
				}
			}

			max := call.GetParam("max")
			if max != "" {
				f, err := strconv.ParseFloat(max, 64)
				if err != nil {
					log.Error().Err(err).Str("max", max).Msg("failed to parse max")
				} else {
					if f != params.Max {
						l.Float64("max", f)
						change = true
					}

					params.Max = f
				}
			}

			if min > max {
				params.Max, params.Min = params.Min, params.Max
			}

			offset := call.GetParam("offset")
			if offset != "" {
				d, err := time.ParseDuration(offset)
				if err != nil {
					log.Error().Err(err).Str("offset", offset).Msg("failed to parse offset")
				} else {
					if d != params.Offset {
						l.Dur("offset", d)
						change = true
					}

					params.Offset = d
				}
			}

			const (
				trueString  = "true"
				falseString = "false"
			)

			alt := call.GetParam("preferAlt")
			if alt == trueString {
				if !params.PreferAlt {
					l.Bool("preferAlt", true)
					change = true
				}

				params.PreferAlt = true
			} else if alt == falseString {
				if params.PreferAlt {
					l.Bool("preferAlt", false)
					change = true
				}

				params.PreferAlt = false
			}

			soft := call.GetParam("preferSoft")
			if soft == trueString {
				if !params.PreferSoft {
					l.Bool("preferSoft", true)
					change = true
				}

				params.PreferSoft = true
			} else if soft == falseString {
				if params.PreferSoft {
					l.Bool("preferSoft", false)
					change = true
				}

				params.PreferSoft = false
			}

			hard := call.GetParam("preferHard")
			if hard == trueString {
				if !params.PreferHard {
					l.Bool("preferHard", true)
					change = true
				}

				params.PreferHard = true
			} else if hard == falseString {
				if params.PreferHard {
					l.Bool("preferHard", false)
					change = true
				}

				params.PreferHard = false
			}

			if change {
				l.Msg("set params")
			}

			respond(w, http.StatusOK, "")
		case "render": // output
			if loadedScripts == nil {
				_, err = w.Write([]byte("no loaded script"))
				if err != nil {
					panic(err)
				}

				return
			}

			for _, script := range loadedScripts.scripts {
				if script.name != "stroke" {
					continue
				}

				err = renderFunscriptHeatmap(*script, call.GetParam("output"))
				if err != nil {
					panic(err)
				}
			}

			respond(w, http.StatusOK, "render")
		case "pause": // seek
			if tcode == nil {
				respond(w, http.StatusInternalServerError, "file not loaded")

				return
			}

			respond(w, http.StatusOK, "pause")

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
		case "play": // seek
			if tcode == nil {
				respond(w, http.StatusInternalServerError, "file not loaded")

				return
			}

			respond(w, http.StatusOK, "play")

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
		default:
			log.Debug().Msgf("%s %s %s", r.RemoteAddr, call.MethodName, call.Params)
		}
	})

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil {
			panic(err)
		}

		os.Exit(1)
	}()

	<-closeChan
}
