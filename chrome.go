package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type chromeStyle struct {
	kind  string // "mac" or "browser"
	dark  bool
	title string
	url   string
}

var (
	trafficRed    = color.RGBA{0xFF, 0x5F, 0x57, 0xFF}
	trafficYellow = color.RGBA{0xFE, 0xBC, 0x2E, 0xFF}
	trafficGreen  = color.RGBA{0x28, 0xC8, 0x40, 0xFF}
)

// addChrome stacks a window title bar above content and returns the composite.
// scale keeps the bar proportional on 2x/3x captures instead of a hairline.
func addChrome(content image.Image, st chromeStyle, scale float64) *image.RGBA {
	b := content.Bounds()
	w := b.Dx()

	barH := int(math.Round(38 * scale))
	if st.kind == "browser" {
		barH = int(math.Round(48 * scale))
	}

	out := image.NewRGBA(image.Rect(0, 0, w, b.Dy()+barH))

	barBG := color.RGBA{0xEC, 0xEC, 0xEE, 0xFF}
	fg := color.RGBA{0x55, 0x55, 0x5C, 0xFF}
	pill := color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	hair := color.RGBA{0x00, 0x00, 0x00, 0x1A}
	if st.dark {
		barBG = color.RGBA{0x2C, 0x2C, 0x2E, 0xFF}
		fg = color.RGBA{0xA8, 0xA8, 0xAE, 0xFF}
		pill = color.RGBA{0x3A, 0x3A, 0x3C, 0xFF}
		hair = color.RGBA{0xFF, 0xFF, 0xFF, 0x14}
	}

	draw.Draw(out, image.Rect(0, 0, w, barH), image.NewUniform(barBG), image.Point{}, draw.Src)
	draw.Draw(out, image.Rect(0, barH-1, w, barH), image.NewUniform(hair), image.Point{}, draw.Over)
	draw.Draw(out, image.Rect(0, barH, w, out.Bounds().Dy()), content, b.Min, draw.Src)

	// Traffic lights, vertically centred in the bar.
	r := 6.0 * scale
	cy := float64(barH) / 2
	x := 20 * scale
	for _, c := range []color.RGBA{trafficRed, trafficYellow, trafficGreen} {
		fillCircle(out, x, cy, r, c)
		x += 20 * scale
	}

	switch st.kind {
	case "mac":
		if st.title != "" {
			drawCenteredText(out, st.title, image.Rect(0, 0, w, barH), 13*scale, fg)
		}
	case "browser":
		// Address pill shares the row with the traffic lights (compact Safari look).
		pillH := int(math.Round(28 * scale))
		pillTop := (barH - pillH) / 2
		left := int(math.Round(x + 12*scale))
		right := w - int(math.Round(20*scale))
		if right > left+pillH {
			fillRoundRect(out, image.Rect(left, pillTop, right, pillTop+pillH), float64(pillH)/2, pill)
			if st.url != "" {
				drawTextAt(out, st.url, float64(left)+14*scale, float64(barH)/2, 12*scale, fg)
			}
		}
	}
	return out
}

func newFace(px float64) (font.Face, error) {
	f, err := opentype.Parse(gomedium.TTF)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingFull})
}

func drawCenteredText(dst *image.RGBA, s string, box image.Rectangle, px float64, c color.RGBA) {
	face, err := newFace(px)
	if err != nil {
		return
	}
	defer face.Close()
	adv := font.MeasureString(face, s)
	x := float64(box.Min.X+box.Dx()/2) - float64(adv)/128
	drawTextAt(dst, s, x, float64(box.Min.Y+box.Dy()/2), px, c)
}

// drawTextAt draws s with (x, midY) as the left edge / vertical centre.
func drawTextAt(dst *image.RGBA, s string, x, midY, px float64, c color.RGBA) {
	face, err := newFace(px)
	if err != nil {
		return
	}
	defer face.Close()
	m := face.Metrics()
	baseline := midY + (float64(m.Ascent)/64-float64(m.Descent)/64)/2
	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(baseline * 64)},
	}
	d.DrawString(s)
}
