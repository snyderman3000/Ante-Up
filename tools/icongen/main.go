// Draws Ante Up's 74×74 menu icon: a red/blue poker chip with an up arrow.
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func main() {
	const S = 74
	img := image.NewRGBA(image.Rect(0, 0, S, S))
	c := float64(S-1) / 2
	red := color.RGBA{0xfe, 0x5f, 0x55, 255}
	blue := color.RGBA{0x00, 0x9d, 0xff, 255}
	white := color.RGBA{0xf4, 0xf4, 0xf4, 255}
	dark := color.RGBA{0x1e, 0x2b, 0x30, 255}
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			dx, dy := float64(x)-c, float64(y)-c
			d := math.Hypot(dx, dy)
			a := math.Atan2(dy, dx)
			var col color.RGBA
			switch {
			case d > 35:
				continue
			case d > 32:
				col = dark
			case d > 23:
				seg := int(math.Floor((a + math.Pi) / (math.Pi / 4)))
				if seg%2 == 0 {
					col = red
				} else {
					col = white
				}
			case d > 21:
				col = dark
			default:
				col = blue
			}
			img.Set(x, y, col)
		}
	}
	// up arrow in the centre
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			fx, fy := float64(x)-c, float64(y)-c
			head := fy >= -14 && fy <= -1 && math.Abs(fx) <= (fy+14)*0.85
			stem := fy > -2 && fy <= 13 && math.Abs(fx) <= 4
			if head || stem {
				img.Set(x, y, white)
			}
		}
	}
	png.Encode(os.Stdout, img)
}
