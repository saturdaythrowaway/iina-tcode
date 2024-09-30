package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func WriteImageFromTcode(tcode *TCode, filename string) error {
	w := 2048 * 16
	h := 512

	img := image.NewRGBA(image.Rect(0, 0, w, h))

	ch := tcode.channels[0]

	for i := range w {
		pos := float64(i+1) / float64(w) * float64(ch.duration)

		pred := PointFromSpline(ch.spline, pos, params.Min, params.Max)
		y := int(float64(h) * pred)

		img.Set(i, int(float64(h)*0.25), color.RGBA{255, 255, 0, 128})
		img.Set(i, int(float64(h)*0.75), color.RGBA{255, 255, 0, 128})
		img.Set(i, int(float64(h)*0.50), color.RGBA{255, 0, 0, 128})

		img.Set(i, int(float64(h)-float64(h)*params.Min/100.0), color.RGBA{0, 255, 255, 128})
		img.Set(i, int(float64(h)*params.Max/100.0), color.RGBA{0, 0, 255, 128})

		img.Set(i, h-y, color.RGBA{255, 255, 255, 255})

	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}
