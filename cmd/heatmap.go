package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"sort"

	"github.com/pkg/errors"
)

var heatmap = []color.RGBA{
	{0, 0, 0, 255},
	{30, 144, 255, 255},
	{34, 139, 34, 255},
	{255, 215, 0, 255},
	{220, 20, 60, 255},
	{147, 112, 219, 255},
	{37, 22, 122, 255},
}

func getSpeed(a1, a2 FunscriptAction) float64 {
	if a1.At == a2.At {
		return 0
	}

	if a2.At < a1.At {
		a1, a2 = a2, a1
	}

	delta := math.Abs(float64(a2.Pos - a1.Pos))

	return (delta / float64(a2.At-a1.At)) * 1000
}

func getLerpedColor(c1, c2 color.RGBA, t float64) color.RGBA {
	c3 := color.RGBA{
		R: uint8(float64(c1.R) + (float64(c2.R)-float64(c1.R))*t),
		G: uint8(float64(c1.G) + (float64(c2.G)-float64(c1.G))*t),
		B: uint8(float64(c1.B) + (float64(c2.B)-float64(c1.B))*t),
		A: uint8(float64(c1.A) + (float64(c2.A)-float64(c1.A))*t),
	}

	return c3
}

func getColor(intensity int) color.RGBA {
	const stepSize = 120.0

	if intensity <= 0 {
		return heatmap[0]
	}

	if intensity > 5*stepSize {
		return heatmap[6]
	}

	intensity += stepSize / 2.0

	d := int(math.Floor(float64(intensity) / stepSize))
	c1 := heatmap[d]
	c2 := heatmap[d+1]
	t := math.Min(1.0, math.Max(0.0, (float64(intensity)-math.Floor(float64(intensity)/stepSize)*stepSize)/stepSize))

	return getLerpedColor(c1, c2, t)
}

func renderFunscriptHeatmap(script Funscript, destination string) error {
	var (
		yWindowSize = 15
		xWindowSize = 50

		height, width int64

		yMin  int
		yMax  int
		lastX int
	)

	width = 512
	height = 32

	background := color.RGBA{0, 0, 0, 0}
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	draw.Draw(img, img.Bounds(), &image.Uniform{background}, image.Point{}, draw.Src)
	msToX := float64(width) / float64(script.Actions[len(script.Actions)-1].At)

	intensityList := make([]int, 0)
	colorAverageList := make([]color.RGBA, 0)
	posList := make([]int, 0)

	yMaxList := make([]int, 0)
	yMaxList = append(yMaxList, script.Actions[0].Pos)

	yMinList := make([]int, 0)
	yMinList = append(yMinList, script.Actions[0].Pos)

	for i := 1; i < len(script.Actions); i++ {
		action := script.Actions[i]
		x := int(math.Floor(msToX * float64(action.At)))
		intensity := int(getSpeed(script.Actions[i-1], script.Actions[i]))

		intensityList = append(intensityList, intensity)
		colorAverageList = append(colorAverageList, getColor(intensity))
		posList = append(posList, action.Pos)

		if len(intensityList) > xWindowSize {
			intensityList = intensityList[1:]
		}

		if len(colorAverageList) > xWindowSize {
			colorAverageList = colorAverageList[1:]
		}

		if len(posList) > yWindowSize {
			posList = posList[1:]
		}

		averageIntensity := 0
		for _, n := range intensityList {
			averageIntensity += n
		}
		averageIntensity /= len(intensityList)

		averageColor := getColor(averageIntensity)

		sortedPos := make([]int, len(posList))
		copy(sortedPos, posList)
		sort.Ints(sortedPos)

		l := (len(sortedPos) / 2) + 1

		bottomHalf := sortedPos[:l]
		topHalf := sortedPos[l-1:]

		averageBottom := 0
		for _, n := range bottomHalf {
			averageBottom += n
		}
		averageBottom /= len(bottomHalf)

		averageTop := 0
		for _, n := range topHalf {
			averageTop += n
		}
		averageTop /= len(topHalf)

		yMaxList = append(yMaxList, action.Pos)
		yMinList = append(yMinList, action.Pos)

		yMax = yMaxList[0]
		for _, n := range yMaxList {
			if n > yMax {
				yMax = n
			}
		}

		yMin = yMinList[0]
		for _, n := range yMinList {
			if n < yMin {
				yMin = n
			}
		}

		if len(yMaxList) > yWindowSize {
			yMaxList = yMaxList[1:]
		}

		if len(yMinList) > yWindowSize {
			yMinList = yMinList[1:]
		}

		y2 := int(height) - int(float64(height)*float64(averageBottom)/100.0)
		y1 := int(height) - int(float64(height)*float64(averageTop)/100.0)

		rect := image.Rect(lastX, y1, x, y2)
		draw.Draw(img, rect, &image.Uniform{averageColor}, image.Point{}, draw.Src)

		lastX = x
	}

	f, err := os.Create(destination)
	if err != nil {
		return errors.Wrap(err, "failed to create heatmap file")
	}

	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		return errors.Wrap(err, "failed to encode image")
	}

	return nil
}
