package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gonum.org/v1/gonum/interp"
)

// https://github.com/multiaxis/tcode-spec

type FunscriptAction struct {
	At  int `json:"at"`
	Pos int `json:"pos"`
}

type Funscript struct {
	path string // for debugging

	Actions  []FunscriptAction `json:"actions"`
	Inverted bool              `json:"inverted"`
	Range    int               `json:"range"`
	Version  string            `json:"version"`
}

type Script struct {
	name    string // for debugging
	Axis    Axis
	Channel int

	Default *Funscript
	Alt     *Funscript
	Soft    *Funscript
	Hard    *Funscript
}

type Scripts struct {
	useSoft bool
	useHard bool
	useAlt  bool

	Stroke  Script // L0
	Surge   Script // L1
	Sway    Script // L2
	Suck    Script // L3/A1
	Twist   Script // R0
	Roll    Script // R1
	Pitch   Script // R2
	Vibrate Script // V0
	Pump    Script // V1/A2
	Valve   Script // V2
}

func (s Scripts) Loaded() []string {
	loaded := []string{}
	for _, script := range []*Script{
		&s.Stroke,
		&s.Surge,
		&s.Sway,
		&s.Suck,
		&s.Twist,
		&s.Roll,
		&s.Pitch,
		&s.Vibrate,
		&s.Pump,
		&s.Valve,
	} {
		if script == nil {
			continue
		}

		path := ""
		label := script.name

		if s.useSoft && script.Soft != nil {
			path = script.Soft.path
			label = "soft." + label
		} else if s.useHard && script.Hard != nil {
			path = script.Hard.path
			label = "hard." + label
		} else if s.useAlt && script.Alt != nil {
			path = script.Alt.path
			label = "alt." + label
		} else {
			if script.Default == nil {
				continue
			}

			path = script.Default.path
		}

		loaded = append(loaded, fmt.Sprintf("%v (%v)", path, label))
	}

	return loaded
}

func (s *Scripts) Reset() {
	s.useSoft = false
	s.useHard = false
	s.useAlt = false

	s.Stroke = Script{}
	s.Surge = Script{}
	s.Sway = Script{}
	s.Suck = Script{}
	s.Twist = Script{}
	s.Roll = Script{}
	s.Pitch = Script{}
	s.Vibrate = Script{}
	s.Pump = Script{}
	s.Valve = Script{}
}

func (s *Scripts) Load(dir string) error {
	s.Reset()

	if dir == "" {
		return errors.New("no folder or dir param")
	}

	if _, err := os.Stat(dir); err != nil {
		return errors.Wrap(err, "failed to stat dir")
	}

	dirents, err := os.ReadDir(dir)
	if err != nil {
		return errors.Wrap(err, "failed to read dir")
	}

	// todo: handle script collisions
	for _, dirent := range dirents {
		if dirent.IsDir() {
			continue
		}

		if !strings.HasSuffix(dirent.Name(), ".funscript") {
			continue
		}

		var name = strings.TrimSuffix(dirent.Name(), ".funscript")

		var soft, hard, alt bool
		switch {
		case strings.HasSuffix(name, ".soft"):
			soft = true
			name = strings.TrimSuffix(name, ".soft")
		case strings.HasSuffix(name, ".hard"):
			hard = true
			name = strings.TrimSuffix(name, ".hard")
		case strings.HasSuffix(name, ".alt"):
			alt = true
			name = strings.TrimSuffix(name, ".alt")
		}

		var script *Script

		switch {
		case strings.HasSuffix(name, ".surge"):
			script = &s.Surge
			script.name = "surge"
			script.Axis = AxisLinear
			script.Channel = 1
		case strings.HasSuffix(name, ".sway"):
			script = &s.Sway
			script.name = "sway"
			script.Axis = AxisLinear
			script.Channel = 2
		case strings.HasSuffix(name, ".stroke"):
			script = &s.Stroke
			script.name = "stroke"
			script.Axis = AxisLinear
			script.Channel = 0
		case strings.HasSuffix(name, ".suck"):
			script = &s.Suck
			script.name = "suck"
			script.Axis = AxisAlt
			script.Channel = 1
		case strings.HasSuffix(name, ".twist"):
			script = &s.Twist
			script.name = "twist"
			script.Axis = AxisRotary
			script.Channel = 0
		case strings.HasSuffix(name, ".roll"):
			script = &s.Roll
			script.name = "roll"
			script.Axis = AxisRotary
			script.Channel = 1
		case strings.HasSuffix(name, ".pitch"):
			script = &s.Pitch
			script.name = "pitch"
			script.Axis = AxisRotary
			script.Channel = 2
		case strings.HasSuffix(name, ".vibrate"):
			script = &s.Vibrate
			script.name = "vibrate"
			script.Axis = AxisVibrate
			script.Channel = 0
		case strings.HasSuffix(name, ".pump"):
			script = &s.Pump
			script.name = "pump"
			script.Axis = AxisAlt
			script.Channel = 2
		case strings.HasSuffix(name, ".valve"):
			script = &s.Valve
			script.name = "valve"
			script.Axis = AxisVibrate
			script.Channel = 2
		default:
			script = &s.Stroke
			script.name = "stroke"
			script.Axis = AxisLinear
			script.Channel = 0
		}

		f, err := parse(filepath.Join(dir, dirent.Name()))
		if err != nil {
			return errors.Wrap(err, "failed to parse funscript")
		}

		if soft {
			script.Soft = f
		} else if hard {
			script.Hard = f
		} else if alt {
			script.Alt = f
		} else {
			script.Default = f
		}
	}

	log.Info().Msgf("loaded %s", dir)

	return nil
}

func (s *Scripts) TCode(p Params) (*TCode, error) {
	// todo: support multi axis

	if s == nil {
		return nil, errors.New("no scripts loaded")
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

	for _, script := range []*Script{
		&s.Stroke,
		&s.Surge,
		&s.Sway,
		&s.Suck,
		&s.Twist,
		&s.Roll,
		&s.Pitch,
		&s.Vibrate,
		&s.Pump,
		&s.Valve,
	} {
		if script == nil {
			continue
		}

		ch := channel{}
		ch.axis = script.Axis
		ch.channel = script.Channel
		ch.spline = &interp.NaturalCubic{}

		var data *Funscript

		if s.useSoft && script.Soft != nil {
			data = script.Soft
		} else if s.useHard && script.Hard != nil {
			data = script.Hard
		} else if s.useAlt && script.Alt != nil {
			data = script.Alt
		} else {
			if script.Default == nil {
				continue
			}

			data = script.Default
		}

		xs, ys := []float64{}, []float64{}

		skip := 0

		for i, action := range data.Actions {
			if action.Pos == 0 || action.Pos == 100 {
				skip = i + 1
				continue
			}

			break
		}

		for _, action := range data.Actions[skip:] {
			xs = append(xs, float64(action.At))

			pos := float64(action.Pos) / 100.0
			pos = p.Min + (pos * (p.Max - p.Min))
			ys = append(ys, pos)
		}

		err := ch.spline.Fit(xs, ys)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fit spline")
		}

		tcode.channels = append(tcode.channels, ch)
	}

	log.Info().Any("loaded", s.Loaded()).Msgf("loaded %d channels", len(tcode.channels))

	return tcode, nil
}

func parse(file string) (*Funscript, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open funscript")
	}

	defer f.Close()

	var fs Funscript
	err = json.NewDecoder(f).Decode(&fs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode funscript")
	}

	fs.path = file

	return &fs, nil
}
