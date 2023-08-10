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
	var template = `<?xml version="1.0"?>
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
func listen(port int) error {
	var loadedScripts *Scripts
	var tcode *TCode

	var params = Params{
		Min: 0.15,
		Max: 0.75,

		Offset: time.Duration(0),

		PreferSoft: false,
		PreferHard: false,
		PreferAlt:  false,
	}

	close := make(chan bool)

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
			close <- true
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

			respond(w, http.StatusOK, fmt.Sprintf("loaded %v", loadedScripts.Loaded()))

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
			min := call.GetParam("min")
			if min != "" {
				f, err := strconv.ParseFloat(min, 64)
				if err != nil {
					log.Error().Err(err).Str("min", min).Msg("failed to parse min")
				} else {
					params.Min = f
				}
			}

			max := call.GetParam("max")
			if max != "" {
				f, err := strconv.ParseFloat(max, 64)
				if err != nil {
					log.Error().Err(err).Str("max", max).Msg("failed to parse max")
				} else {
					params.Max = f
				}
			}

			if min > max {
				params.Min = params.Max
			}

			offset := call.GetParam("offset")
			if offset != "" {
				d, err := time.ParseDuration(offset)
				if err != nil {
					log.Error().Err(err).Str("offset", offset).Msg("failed to parse offset")
				} else {
					params.Offset = d
				}
			}

			alt := call.GetParam("preferAlt")
			if alt == "true" {
				params.PreferAlt = true
			} else if alt == "false" {
				params.PreferAlt = false
			}

			soft := call.GetParam("preferSoft")
			if soft == "true" {
				params.PreferSoft = true
			} else if soft == "false" {
				params.PreferSoft = false
			}

			hard := call.GetParam("preferHard")
			if hard == "true" {
				params.PreferHard = true
			} else if hard == "false" {
				params.PreferHard = false
			}

			res := []string{
				"set",
				"min=" + fmt.Sprintf("%f", params.Min),
				"max=" + fmt.Sprintf("%f", params.Max),
				"offset=" + params.Offset.String(),
				"preferAlt=" + fmt.Sprintf("%t", params.PreferAlt),
				"preferSoft=" + fmt.Sprintf("%t", params.PreferSoft),
				"preferHard=" + fmt.Sprintf("%t", params.PreferHard),
			}

			respond(w, http.StatusOK, strings.Join(res, " "))
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

	<-close

	return nil
}
