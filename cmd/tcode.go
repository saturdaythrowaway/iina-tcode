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
	Axis    Axis
	Channel int
	Value   float64
}

func (tm TCodeMessage) String() string {
	if tm.Axis == "" {
		return ""
	}

	var pos string

	switch tm.Value {
	case 0.0:
		pos = "00000"
	case 1.0:
		pos = "99999"
	default:
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

func NewTCode() *TCode {
	return &TCode{
		ts:     0,
		ticker: time.NewTicker(TPS),
	}
}

func (t *TCode) Pause() {
	if t == nil {
		return
	}

	t.Zero()

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

	messages := make(chan string)

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

				if params.Max == 0 || params.Max > 1 {
					params.Max = 1
				}

				if params.Min < 0 || params.Min == 1 {
					params.Min = 0
				}

				if params.Min > params.Max {
					params.Min = params.Max
				}

				strokeRange := (params.Max - params.Min)
				pos := opos*strokeRange + params.Min

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

func (t *TCode) setValue(value float64) {
	if t == nil {
		return
	}

	for _, c := range t.channels {
		err := sendTCode((TCodeMessage{
			Axis:    c.axis,
			Channel: c.channel,
			Value:   value,
		}).String())
		if err != nil {
			log.Error().Err(err).Msg("failed to send tcode")
		}
	}
}

func (t *TCode) Zero() {
	t.setValue(0.0)
}

func (t *TCode) Center() {
	t.setValue(0.5)
}

func (t *TCode) Max() {
	t.setValue(1.0)
}

func (t *TCode) Reset() {
	defer func() {
		_ = recover() // don't panic if channel is closed
	}()

	if t.messages != nil {
		close(t.messages)
	}

	t.channels = nil
	t.messages = nil

}

func (t *TCode) Close() {
	t.Reset()

	log.Info().Msg("closing")

	t.Max()
	t.Zero()
	t.Center()
}
