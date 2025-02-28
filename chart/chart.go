package chart

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/fogleman/gg"
)

type Chart struct {
	text   string
	data   []float64
	colors []color.Color

	lines1 []int
	lines2 []int

	Areas []Area
}

type Area struct {
	Index1 int
	Index2 int
	Good   bool
}

func NewChart() *Chart {
	var c Chart
	return &c
}

func (c *Chart) SetData(data []float64) {
	c.data = data
	c.colors = make([]color.Color, 0)
	for i := 0; i < len(data); i++ {
		c.colors = append(c.colors, color.RGBA{0, 150, 250, 255})
	}
}

func (c *Chart) SetText(text string) {
	c.text = text
}

func (c *Chart) SetColors(colors []color.Color) {
	c.colors = colors
}

func (c *Chart) SetLines1(lines []int) {
	c.lines1 = lines
}

func (c *Chart) SetLines2(lines []int) {
	c.lines2 = lines
}

func (c *Chart) DrawTrace() image.Image {
	minValue := c.data[0]
	maxValue := c.data[0]
	for _, v := range c.data {
		if v < minValue {
			minValue = v
		}
		if v > maxValue {
			maxValue = v
		}
	}

	// pixel size = 2
	width := len(c.data) * 2
	height := 200.0

	dc := gg.NewContext(width, int(height))
	dc.SetRGB(0.01, 0.1, 0.1)
	dc.Clear()

	// draw grid 20 px
	/*dc.SetRGB(0.5, 0.8, 0.5)
	dc.SetLineWidth(0.2)
	for i := 0; i < int(width); i += 20 {
		dc.MoveTo(float64(i), 0)
		dc.LineTo(float64(i), float64(width))
	}
	dc.Stroke()*/

	dc.SetRGB(0, 0.5, 0.9)
	dc.SetLineWidth(1)

	for i := 0; i < len(c.data); i++ {
		x := float64(i) / float64(len(c.data)) * float64(width)
		y := (c.data[i] - minValue) / (maxValue - minValue) * float64(height)

		if i >= len(c.colors) {
			dc.SetRGB(0, 0.5, 0.9)
		} else {
			dc.SetColor(c.colors[i])
		}

		if i == 0 {
			dc.MoveTo(x, height-y)
		} else {
			dc.LineTo(x, height-y)
		}
	}
	dc.Stroke()

	// draw zero line
	dc.SetRGB(0.5, 0.5, 0.5)
	dc.SetLineWidth(1)
	y := (0 - minValue) / (maxValue - minValue) * float64(height)
	dc.MoveTo(0, height-y)
	dc.LineTo(float64(width), height-y)
	dc.Stroke()

	// draw bottom line
	dc.SetRGB(0.2, 0.2, 0.2)
	dc.SetLineWidth(1)
	dc.MoveTo(0, height)
	dc.LineTo(float64(width), height)
	dc.Stroke()

	// draw vertical lines
	dc.SetRGB(0.0, 0.4, 0.0)
	dc.SetLineWidth(1)
	for _, l := range c.lines1 {
		x := float64(l) / float64(len(c.data)) * float64(width)
		dc.MoveTo(x, 0)
		dc.LineTo(x, height)
		dc.Stroke()
	}

	dc.SetRGB(0.0, 0.4, 0.0)
	dc.SetLineWidth(1)
	for _, l := range c.lines2 {
		x := float64(l) / float64(len(c.data)) * float64(width)
		dc.MoveTo(x, 0)
		dc.LineTo(x, height)
		dc.Stroke()
	}

	// draw areas
	for _, a := range c.Areas {
		if a.Good {
			dc.SetRGBA(0.0, 0.8, 0.0, 0.2)
		} else {
			dc.SetRGBA(0.8, 0.0, 0.0, 0.2)
		}

		x1 := float64(a.Index1) / float64(len(c.data)) * float64(width)
		x2 := float64(a.Index2) / float64(len(c.data)) * float64(width)
		dc.DrawRectangle(x1, 0, x2-x1, height)
		dc.Fill()
	}

	dc.SetRGB(1, 1, 1)
	dc.DrawString("["+c.text+"]", 10, 10)

	img := image.NewRGBA(image.Rect(0, 0, width, int(height)))
	draw.Draw(img, img.Bounds(), dc.Image(), image.Point{0, 0}, draw.Src)

	return img
}

func CombineImages(images ...image.Image) image.Image {
	width := 0
	height := 0
	for _, img := range images {
		if img.Bounds().Dx() > width {
			width = img.Bounds().Dx()
		}
		height += img.Bounds().Dy()
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	y := 0
	for _, img := range images {
		draw.Draw(dst, img.Bounds().Add(image.Pt(0, y)), img, image.Point{0, 0}, draw.Src)
		y += img.Bounds().Dy()
	}

	return dst
}
