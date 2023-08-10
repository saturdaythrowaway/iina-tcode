package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type Axis string

const (
	AxisLinear  Axis = "L"
	AxisRotary  Axis = "R"
	AxisVibrate Axis = "V"
	AxisAlt     Axis = "A"
)

type TCodeMessage struct {
	At      int
	Axis    Axis
	Channel int
	Value   float64
}

func (tm TCodeMessage) String() string {
	if tm.Axis == "" {
		return ""
	}

	pos := ""
	if tm.Value == 1.0 {
		pos = "99999"
	} else if tm.Value == 0.0 {
		pos = "00000"
	} else {
		pos = fmt.Sprintf("%f", tm.Value)[2:]
		if len(pos) > 5 {
			pos = pos[:5]
		} else if len(pos) < 5 {
			pos = fmt.Sprintf("%05s", pos)
		}
	}

	return fmt.Sprintf("%s%d%s", tm.Axis, tm.Channel, pos)
}

type TCode struct {
	channels []channel

	params Params

	messages chan string
	ts       time.Duration
	ticker   *time.Ticker
}

type spline interface {
	Predict(x float64) float64
	Fit(x, y []float64) error
}

type channel struct {
	axis    Axis
	channel int
	spline  spline
}

type Params struct {
	Min, Max float64

	Offset time.Duration

	PreferSoft bool
	PreferHard bool
	PreferAlt  bool
}

func NewTCode(p Params) *TCode {
	return &TCode{
		ts:     0,
		ticker: time.NewTicker(TPS),
		params: p,
	}
}

func (t *TCode) Pause() {
	if t == nil {
		return
	}

	log.Debug().Msg("pause")

	t.ticker.Reset(math.MaxInt64)
}

func (t *TCode) Play() {
	if t == nil {
		return
	}

	log.Debug().Msg("play")

	t.ticker.Reset(TPS)
}

func (t *TCode) Seek(seek time.Duration) {
	if t == nil {
		return
	}

	log.Trace().Dur("seek", seek).Msg("seek")

	t.ts = seek
}

func (t *TCode) Tick() <-chan string {
	if t == nil {
		return nil
	}

	var messages = make(chan string)

	t.messages = messages

	go func() {
		defer func() {
			_ = recover() // don't panic if channel is closed
		}()

		last := ""

		t.ticker.Reset(TPS)

		for range t.ticker.C {
			var messages []string

			for _, c := range t.channels {
				if c.spline == nil {
					continue
				}

				opos := c.spline.Predict(float64(t.ts.Milliseconds())) / 100.0

				if opos < 0.0 {
					opos = 0.0
				}

				if opos > 1.0 {
					opos = 1.0
				}

				strokeRange := (t.params.Max - t.params.Min)
				pos := opos*strokeRange + t.params.Min

				msg := TCodeMessage{
					Axis:    c.axis,
					Channel: c.channel,
					Value:   pos,
				}

				messages = append(messages, msg.String())
			}

			t.ts += TPS

			if len(messages) == 0 {
				continue
			}

			msg := strings.Join(messages, ", ")

			if msg == last {
				log.Trace().Str("tcode", msg).Msg("skip duplicate")
				continue
			}

			last = msg
			t.messages <- last
		}
	}()

	return messages
}

func (t *TCode) Close() {
	defer func() {
		_ = recover() // don't panic if channel is closed
	}()

	if t.messages != nil {
		close(t.messages)
	}

	t.channels = nil
	t.messages = nil

	log.Info().Msg("closing")
}
