package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"gonum.org/v1/gonum/interp"
)

var tc *TCode

type Axis string

const (
	AxisLinear  Axis = "L"
	AxisRotary  Axis = "R"
	AxisVibrate Axis = "V"
	AxisAlt     Axis = "A"
)

type TCodeMessage struct {
	Axis     Axis
	Channel  int
	Value    float64
	Duration time.Duration
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

	if tm.Duration == 0 {
		return fmt.Sprintf("%s%d%s", tm.Axis, tm.Channel, pos)
	}

	return fmt.Sprintf("%s%d%sI%d", tm.Axis, tm.Channel, pos, tm.Duration.Milliseconds())
}

type TCode struct {
	channels []channel

	messages chan string
	ts       time.Duration
	ticker   *time.Ticker
}

type spline interface {
	interp.Predictor
	interp.DerivativePredictor

	Fit(xs, ys []float64) error
}

func PointFromSpline(s spline, pos float64, bottom, top float64) float64 {
	d := 100.0
	v := d + (bottom+top)*2

	point := (s.Predict(pos) / v) + (bottom / d)

	if point < 0.0 {
		return 0.0
	}

	if point > 1.0 {
		return 1.0
	}

	return point
}

type channel struct {
	axis     Axis
	channel  int
	spline   spline
	duration int

	maxOffset int
	minOffset int
}

func NewTCode() *TCode {
	tc = &TCode{
		ts:     0,
		ticker: time.NewTicker(TPS),
	}

	return tc
}

func (t *TCode) Pause() {
	if t == nil {
		return
	}

	t.setValue(params.Min, time.Second)

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

				pos := PointFromSpline(c.spline, float64(t.ts.Milliseconds()), params.Min, params.Max)
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

func (t *TCode) setValue(value float64, duration time.Duration) {
	if t == nil {
		return
	}

	for _, c := range t.channels {
		err := sendTCode((TCodeMessage{
			Axis:     c.axis,
			Channel:  c.channel,
			Value:    value,
			Duration: duration,
		}).String())
		if err != nil {
			log.Error().Err(err).Msg("failed to send tcode")
		}
	}
}

func (t *TCode) Stop() {
	if t == nil {
		return
	}

	t.ticker.Reset(math.MaxInt64)
}

func (t *TCode) Reset() {
	defer func() {
		_ = recover() // don't panic if channel is closed
	}()

	t.Stop()

	if t.messages != nil {
		close(t.messages)
	}

	t.channels = nil
	t.messages = nil
}

func (t *TCode) Close() {
	t.Stop()

	log.Info().Msg("closing")

	dur := time.Second

	t.setValue(0.00001, dur)
	time.Sleep(dur)
	t.setValue(0.99999, dur)
	time.Sleep(dur)
	t.setValue(0.00001, dur)
	time.Sleep(dur)
	t.setValue(0.5, dur/2)
	time.Sleep(dur / 2)
	t.setValue(0.5, 0)
}
