package main

import (
	"image"
	"image/color"
)

func makeCoverImage() *image.RGBA {
	const size = 768
	canvas := image.NewRGBA(image.Rect(0, 0, size, size))
	center := size / 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := x-center, y-center
			distance := dx*dx + dy*dy
			glow := float64(max(0, 150000-distance)) / 150000
			canvas.SetRGBA(x, y, color.RGBA{
				R: uint8(14 + y/96 + int(glow*20)),
				G: uint8(22 + y/80 + int(glow*28)),
				B: uint8(54 + y/32 + int(glow*70)),
				A: 255,
			})
			if (x*37+y*73)%997 < 3 && distance > 120000 {
				fillCircle(canvas, x, y, 1, color.RGBA{R: 129, G: 197, B: 236, A: 255})
			}
		}
	}

	fillCircle(canvas, center, center, 276, color.RGBA{R: 41, G: 59, B: 112, A: 255})
	strokeCircle(canvas, center, center, 276, 3, color.RGBA{R: 92, G: 128, B: 207, A: 255})
	fillPolygon(canvas, []image.Point{{220, 155}, {464, 155}, {548, 239}, {548, 613}, {220, 613}}, color.RGBA{R: 8, G: 12, B: 31, A: 255})
	fillPolygon(canvas, []image.Point{{204, 139}, {448, 139}, {532, 223}, {532, 597}, {204, 597}}, color.RGBA{R: 239, G: 243, B: 255, A: 255})
	fillPolygon(canvas, []image.Point{{448, 139}, {448, 223}, {532, 223}}, color.RGBA{R: 191, G: 207, B: 238, A: 255})
	drawText(canvas, 296, 248, "AGE", 8, color.RGBA{R: 47, G: 64, B: 111, A: 255})
	drawText(canvas, 298, 314, "FILE", 5, color.RGBA{R: 87, G: 103, B: 148, A: 255})
	fillRoundedRect(canvas, 300, 363, 437, 476, 18, color.RGBA{R: 47, G: 190, B: 205, A: 255})
	strokeArc(canvas, 322, 366, 390, 314, 3, color.RGBA{R: 47, G: 190, B: 205, A: 255})
	fillCircle(canvas, 368, 414, 12, color.RGBA{R: 26, G: 46, B: 94, A: 255})
	fillRect(canvas, 362, 419, 374, 443, color.RGBA{R: 26, G: 46, B: 94, A: 255})
	drawText(canvas, 290, 513, "NOSTR", 8, color.RGBA{R: 55, G: 73, B: 122, A: 255})
	fillCircle(canvas, 132, 292, 8, color.RGBA{R: 75, G: 216, B: 210, A: 255})
	fillCircle(canvas, 632, 472, 6, color.RGBA{R: 243, G: 181, B: 106, A: 255})
	return canvas
}

func fillRect(canvas *image.RGBA, left, top, right, bottom int, value color.RGBA) {
	for y := top; y < bottom; y++ {
		for x := left; x < right; x++ {
			canvas.SetRGBA(x, y, value)
		}
	}
}

func fillRoundedRect(canvas *image.RGBA, left, top, right, bottom, radius int, value color.RGBA) {
	for y := top; y < bottom; y++ {
		for x := left; x < right; x++ {
			cx, cy := x, y
			if x < left+radius {
				cx = left + radius
			} else if x >= right-radius {
				cx = right - radius - 1
			}
			if y < top+radius {
				cy = top + radius
			} else if y >= bottom-radius {
				cy = bottom - radius - 1
			}
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= radius*radius {
				canvas.SetRGBA(x, y, value)
			}
		}
	}
}

func fillCircle(canvas *image.RGBA, centerX, centerY, radius int, value color.RGBA) {
	for y := centerY - radius; y <= centerY+radius; y++ {
		for x := centerX - radius; x <= centerX+radius; x++ {
			dx, dy := x-centerX, y-centerY
			if dx*dx+dy*dy <= radius*radius {
				canvas.SetRGBA(x, y, value)
			}
		}
	}
}

func strokeCircle(canvas *image.RGBA, centerX, centerY, radius, width int, value color.RGBA) {
	outer, inner := radius*radius, (radius-width)*(radius-width)
	for y := centerY - radius; y <= centerY+radius; y++ {
		for x := centerX - radius; x <= centerX+radius; x++ {
			dx, dy := x-centerX, y-centerY
			distance := dx*dx + dy*dy
			if distance <= outer && distance >= inner {
				canvas.SetRGBA(x, y, value)
			}
		}
	}
}

func strokeArc(canvas *image.RGBA, left, top, right, bottom, width int, value color.RGBA) {
	centerX := (left + right) / 2
	centerY := (top + bottom) / 2
	radiusX, radiusY := (right-left)/2, (bottom-top)/2
	for y := top; y <= bottom; y++ {
		for x := left; x <= right; x++ {
			dx, dy := float64(x-centerX)/float64(radiusX), float64(y-centerY)/float64(radiusY)
			if y <= centerY && dx*dx+dy*dy <= 1 && dx*dx+dy*dy >= 0.78 {
				for offset := -width / 2; offset <= width/2; offset++ {
					canvas.SetRGBA(x+offset, y, value)
				}
			}
		}
	}
}

func fillPolygon(canvas *image.RGBA, points []image.Point, value color.RGBA) {
	minY, maxY := points[0].Y, points[0].Y
	for _, point := range points[1:] {
		minY = min(minY, point.Y)
		maxY = max(maxY, point.Y)
	}
	for y := minY; y <= maxY; y++ {
		intersections := make([]int, 0, len(points))
		for index, start := range points {
			end := points[(index+1)%len(points)]
			if (start.Y <= y && end.Y > y) || (end.Y <= y && start.Y > y) {
				x := start.X + (y-start.Y)*(end.X-start.X)/(end.Y-start.Y)
				intersections = append(intersections, x)
			}
		}
		for index := 0; index+1 < len(intersections); index += 2 {
			for x := intersections[index]; x <= intersections[index+1]; x++ {
				canvas.SetRGBA(x, y, value)
			}
		}
	}
}

func drawText(canvas *image.RGBA, left, top int, text string, scale int, value color.RGBA) {
	for letterIndex, letter := range text {
		glyph, ok := font5x7[letter]
		if !ok {
			continue
		}
		originX := left + letterIndex*scale*6
		for y, row := range glyph {
			for x := 0; x < 5; x++ {
				if row&(1<<(4-x)) == 0 {
					continue
				}
				fillRect(canvas, originX+x*scale, top+y*scale, originX+(x+1)*scale, top+(y+1)*scale, value)
			}
		}
	}
}

var font5x7 = map[rune][7]uint8{
	'A': {14, 17, 17, 31, 17, 17, 17},
	'G': {14, 17, 16, 23, 17, 17, 15},
	'E': {31, 16, 16, 30, 16, 16, 31},
	'F': {31, 16, 16, 30, 16, 16, 16},
	'I': {31, 4, 4, 4, 4, 4, 31},
	'L': {16, 16, 16, 16, 16, 16, 31},
	'N': {17, 25, 25, 21, 19, 19, 17},
	'O': {14, 17, 17, 17, 17, 17, 14},
	'S': {15, 16, 16, 14, 1, 1, 30},
	'T': {31, 4, 4, 4, 4, 4, 4},
	'R': {30, 17, 17, 30, 20, 18, 17},
}
