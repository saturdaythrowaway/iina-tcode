package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"gonum.org/v1/gonum/interp"
)

// https://github.com/multiaxis/tcode-spec

type FunscriptAction struct {
	At  int `json:"at"`
	Pos int `json:"pos"`
}

type Script struct {
	path     string // for debugging
	name     string // for debugging
	filename string // for debugging

	Axis     Axis      `json:"-"`
	Channel  int       `json:"-"`
	Modifier ScriptMod `json:"-"`

	Actions  []FunscriptAction `json:"actions"`
	Inverted bool              `json:"inverted"`
	Range    int               `json:"range"`
	Version  string            `json:"version"`
}

func (s Script) String() string {
	if s.Modifier != ScriptModDefault {
		return fmt.Sprintf("%s (%s): %s", s.name, s.Modifier, s.filename)
	}

	return fmt.Sprintf("%s: %s", s.name, s.filename)
}

type Scripts struct {
	preferedModifier ScriptMod

	scripts map[string]*Script
}

func NewScript(path string) (*Script, error) {
	var name = strings.TrimSuffix(path, ".funscript")
	script := Script{}
	script.path = path

	switch {
	case strings.HasSuffix(name, ".soft"):
		script.Modifier = ScriptModSoft
	case strings.HasSuffix(name, ".hard"):
		script.Modifier = ScriptModHard
	case strings.HasSuffix(name, ".alt"):
		script.Modifier = ScriptModAlt
	}

	script.filename = filepath.Base(name)

	if script.Modifier != ScriptModDefault {
		name = strings.TrimSuffix(name, "."+script.Modifier.String())
	}

	switch {
	case strings.HasSuffix(name, ".surge"):
		script.name = "surge"
		script.Axis = AxisLinear
		script.Channel = 1
	case strings.HasSuffix(name, ".sway"):
		script.name = "sway"
		script.Axis = AxisLinear
		script.Channel = 2
	case strings.HasSuffix(name, ".stroke"):
		script.name = "stroke"
		script.Axis = AxisLinear
		script.Channel = 0
	case strings.HasSuffix(name, ".suck"):
		script.name = "suck"
		script.Axis = AxisAlt
		script.Channel = 1
	case strings.HasSuffix(name, ".twist"):
		script.name = "twist"
		script.Axis = AxisRotary
		script.Channel = 0
	case strings.HasSuffix(name, ".roll"):
		script.name = "roll"
		script.Axis = AxisRotary
		script.Channel = 1
	case strings.HasSuffix(name, ".pitch"):
		script.name = "pitch"
		script.Axis = AxisRotary
		script.Channel = 2
	case strings.HasSuffix(name, ".vibrate"):
		script.name = "vibrate"
		script.Axis = AxisVibrate
		script.Channel = 0
	case strings.HasSuffix(name, ".pump"):
		script.name = "pump"
		script.Axis = AxisAlt
		script.Channel = 2
	case strings.HasSuffix(name, ".valve"):
		script.name = "valve"
		script.Axis = AxisVibrate
		script.Channel = 2
	default:
		script.name = "stroke"
		script.Axis = AxisLinear
		script.Channel = 0
	}

	f, err := os.Open(script.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open funscript: %w", err)
	}

	err = json.NewDecoder(f).Decode(&script)
	if err != nil {
		f.Close()

		return nil, fmt.Errorf("failed to decode funscript: %w", err)
	}

	f.Close()

	return &script, nil
}

type ScriptMod int

func (s ScriptMod) String() string {
	switch s {
	case ScriptModAlt:
		return "alt"
	case ScriptModSoft:
		return "soft"
	case ScriptModHard:
		return "hard"
	default:
		return ""
	}
}

const (
	ScriptModDefault ScriptMod = iota
	ScriptModAlt
	ScriptModSoft
	ScriptModHard
)

func (s Scripts) Loaded() []string {
	loaded := []string{}
	for _, script := range s.scripts {
		loaded = append(loaded, filepath.Base(script.path))
	}

	return loaded
}

func (s *Scripts) Reset() {
	s.scripts = map[string]*Script{}
}

func (s *Scripts) Load(path string) error {
	s.Reset()

	if path == "" {
		return fmt.Errorf("no folder or dir param")
	}

	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat dir: %w", err)
	}

	dir := ""
	filename := ""
	if fi.IsDir() {
		dir = path
		filename = ""
	} else {
		dir = filepath.Dir(path)
		filename = filepath.Base(path)
	}

	log.Debug().Str("filename", filename).Msgf("loading scripts from %s", dir)

	dirents, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read dir: %w", err)
	}

	availableScripts := map[string][]*Script{}

	// todo: handle script collisions
	for _, dirent := range dirents {
		if dirent.IsDir() {
			continue
		}

		if !strings.HasSuffix(dirent.Name(), ".funscript") {
			continue
		}

		script, err := NewScript(filepath.Join(dir, dirent.Name()))
		if err != nil {
			log.Warn().Err(err).Msgf("failed to load script %s", dirent.Name())

			continue
		}

		fmt.Println(len(script.Actions))

		if _, ok := availableScripts[script.name]; !ok {
			availableScripts[script.name] = []*Script{script}
		} else {
			availableScripts[script.name] = append(availableScripts[script.name], script)
		}
	}

	for _, axis := range availableScripts {
		if len(axis) == 0 {
			continue
		}

		preferedScripts := []*Script{}
		scripts := []*Script{}

		if len(axis) > 1 {
			for _, script := range axis {
				if s.preferedModifier == script.Modifier {
					preferedScripts = append(preferedScripts, script)
				} else {
					scripts = append(scripts, script)
				}
			}

			scripts = append(preferedScripts, scripts...)

			if filename != "" {
				for _, script := range scripts {
					if strings.HasPrefix(filename, script.filename) {
						scripts = []*Script{script}

						break
					}
				}
			}
		} else {
			scripts = axis
		}

		s.scripts[scripts[0].name] = scripts[0]
	}

	if len(s.scripts) == 0 {
		return fmt.Errorf("no scripts loaded")
	}

	scripts := []string{}
	for _, script := range s.scripts {
		scripts = append(scripts, script.String())
	}

	log.Info().Strs("scripts", scripts).Msgf("loaded %s", dir)

	return nil
}

func (s *Scripts) TCode(p Params) (*TCode, error) {
	if s == nil {
		return nil, fmt.Errorf("no scripts loaded")
	}

	if p.Max == 0 || p.Max > 1 {
		p.Max = 1
	}

	if p.Min < 0 {
		p.Min = 0
	}

	if p.Min > p.Max {
		p.Min = p.Max
	}

	tcode := NewTCode(p)
	tcode.channels = make([]channel, 0)

	for _, script := range s.scripts {
		ch := channel{}
		ch.axis = script.Axis
		ch.channel = script.Channel
		ch.spline = &interp.FritschButland{}

		skip := 0

		for i, action := range script.Actions {
			if action.Pos == 0 || action.Pos == 100 {
				skip = i + 1
				continue
			}

			break
		}

		xs := make([]float64, 0, len(script.Actions)-skip)
		ys := make([]float64, 0, len(script.Actions)-skip)

		sort.Slice(script.Actions, func(i, j int) bool {
			return script.Actions[i].At < script.Actions[j].At
		})

		for _, action := range script.Actions[skip:] {
			xs = append(xs, float64(action.At))
			ys = append(ys, float64(action.Pos))
		}

		for i := 0; i < len(xs)-1; i++ {
			if xs[i+1] == xs[i] {
				y := (ys[i] + ys[i+1]) / 2
				ys = append(ys[:i], ys[i+1:]...)
				xs = append(xs[:i], xs[i+1:]...)
				ys[i] = y
				i--
			}
		}

		err := ch.spline.Fit(xs, ys)
		if err != nil {
			return nil, fmt.Errorf("failed to fit spline: %w", err)
		}

		tcode.channels = append(tcode.channels, ch)
	}

	log.Info().Any("loaded", s.Loaded()).Msgf("loaded %d channels", len(tcode.channels))

	err := sendTCode("L00, L10, L20, L30, A10, R00, R10, R20, V00, V10, A20, V20")
	if err != nil {
		return tcode, err
	}

	return tcode, nil
}
