package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"gonum.org/v1/gonum/interp"
)

// https://github.com/multiaxis/tcode-spec

const defaultAxis = "stroke"

var axisMap = map[string]Script{
	"stroke":  {Axis: AxisLinear, Channel: 0},
	"surge":   {Axis: AxisLinear, Channel: 1},
	"sway":    {Axis: AxisLinear, Channel: 2},
	"twist":   {Axis: AxisRotary, Channel: 0},
	"roll":    {Axis: AxisRotary, Channel: 1},
	"pitch":   {Axis: AxisRotary, Channel: 2},
	"vibrate": {Axis: AxisVibrate, Channel: 0},
	"suck":    {Axis: AxisAlt, Channel: 1},
	"pump":    {Axis: AxisAlt, Channel: 2},
	"valve":   {Axis: AxisVibrate, Channel: 2},
}

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

	Duration int `json:"duration"`

	Actions  []FunscriptAction `json:"actions"`
	Inverted any               `json:"inverted"`
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
	name := strings.TrimSuffix(path, ".funscript")
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

	ext := filepath.Ext(script.filename)
	if script.Modifier != ScriptModDefault {
		ext = strings.TrimSuffix(ext, "."+script.Modifier.String())
	}

	if ext != "" {
		ext = ext[1:]
	} else {
		ext = defaultAxis
	}

	if s, ok := axisMap[ext]; ok {
		script.name = ext
		script.Axis = s.Axis
		script.Channel = s.Channel
	} else {
		log.Warn().Str("ext", ext).Msgf("unknown axis")

		s := axisMap[defaultAxis]
		script.name = defaultAxis
		script.Axis = s.Axis
		script.Channel = s.Channel
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
	case ScriptModDefault:
		return ""
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
		return errors.New("no folder or dir param")
	}

	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat dir: %w", err)
	}

	var (
		dir      string
		filename string
	)

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
		return errors.New("no scripts loaded")
	}

	scripts := []string{}
	for _, script := range s.scripts {
		scripts = append(scripts, script.String())
	}

	log.Info().Strs("scripts", scripts).Msgf("loaded %s", dir)

	return nil
}

func (s *Scripts) TCode() (*TCode, error) {
	if s == nil {
		return nil, errors.New("no scripts loaded")
	}

	tcode := NewTCode()
	tcode.channels = make([]channel, 0)

	for _, script := range s.scripts {
		ch := channel{}

		ch.axis = script.Axis
		ch.channel = script.Channel
		ch.spline = &interp.FritschButland{}

		xs := make([]float64, 0, len(script.Actions))
		ys := make([]float64, 0, len(script.Actions))

		sort.Slice(script.Actions, func(i, j int) bool {
			return script.Actions[i].At < script.Actions[j].At
		})

		for _, action := range script.Actions {
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

		if len(xs) == 0 || len(ys) == 0 {
			log.Warn().Msgf("skipping %s: no actions", script)

			continue
		}

		err := ch.spline.Fit(xs, ys)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to fit spline for %s", script)
		}

		tcode.channels = append(tcode.channels, ch)
	}

	log.Info().Any("loaded", s.Loaded()).Msgf("loaded %d channels", len(tcode.channels))

	err := sendTCode("L10, L20, L30, A10, R00, R10, R20, V00, V10, A20, V20")
	if err != nil {
		return tcode, err
	}

	return tcode, nil
}
