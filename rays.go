package main

import (
	"image"
	"math"
)

// Light rays: the soft diagonal shadow a window blind casts across the backdrop.
// Bands of light run parallel at an angle, blurred and unevenly spaced, fading
// out as they travel away from the light source.

type raysOptions struct {
	strength float64 // 0 = off
	angle    float64 // degrees; direction the bands travel
	width    float64 // stretches the band pattern; >1 = fewer, wider bands
}

// stripe is one band of light: centre and width along the projected axis (both
// as a fraction of the sweep), plus a relative brightness.
type stripe struct{ centre, width, intensity float64 }

// Hand-tuned so the bands cluster in pairs the way real blinds do, rather than
// reading as an even, mechanical comb. Widths are fractions of the sweep, kept
// narrow enough that the bands stay legible as bands and not as haze.
var blindStripes = []stripe{
	{0.04, 0.012, 0.75},
	{0.11, 0.024, 1.00},
	{0.19, 0.009, 0.45},
	{0.27, 0.028, 0.95},
	{0.36, 0.011, 0.60},
	{0.44, 0.020, 0.85},
	{0.54, 0.030, 1.00},
	{0.63, 0.010, 0.50},
	{0.71, 0.022, 0.90},
	{0.80, 0.013, 0.60},
	{0.89, 0.026, 0.80},
	{0.97, 0.010, 0.45},
}

// bandValue returns the light intensity at projected position u ∈ [0,1].
func bandValue(u float64) float64 {
	var v float64
	for _, s := range blindStripes {
		d := (u - s.centre) / s.width
		v += s.intensity * math.Exp(-d*d)
	}
	return math.Min(1, v)
}

// applyRays modulates canvas in place. mask, when non-nil, marks pixels the
// rays must skip (the window) so the backdrop alone catches the light.
func applyRays(canvas *image.RGBA, o raysOptions, skip *image.Alpha, skipAt image.Point) {
	if o.strength <= 0 {
		return
	}
	if o.width <= 0 {
		o.width = 1
	}
	b := canvas.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())

	rad := o.angle * math.Pi / 180
	// Bands run perpendicular to this axis.
	nx, ny := math.Cos(rad), math.Sin(rad)

	var lo, hi float64
	for i, c := range [][2]float64{{0, 0}, {w, 0}, {0, h}, {w, h}} {
		p := c[0]*nx + c[1]*ny
		if i == 0 || p < lo {
			lo = p
		}
		if i == 0 || p > hi {
			hi = p
		}
	}
	span := hi - lo
	if span == 0 {
		span = 1
	}

	// The light enters from the corner the bands travel away from, and dies out
	// across the diagonal — otherwise the pattern looks like wallpaper.
	srcX, srcY := 0.0, 0.0
	if nx < 0 {
		srcX = w
	}
	if ny < 0 {
		srcY = h
	}
	maxDist := math.Hypot(w, h)

	const lutN = 2048
	var lut [lutN]float64
	for i := range lut {
		lut[i] = bandValue(float64(i) / (lutN - 1))
	}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if skip != nil {
				sx, sy := x-skipAt.X, y-skipAt.Y
				if image.Pt(sx, sy).In(skip.Bounds()) && skip.Pix[skip.PixOffset(sx, sy)] > 8 {
					continue
				}
			}
			fx, fy := float64(x-b.Min.X), float64(y-b.Min.Y)
			// Dividing by width shows less of the pattern across the same sweep,
			// so each band comes out proportionally wider.
			u := (fx*nx + fy*ny - lo) / span / o.width
			u -= math.Floor(u) // the pattern tiles, so wide settings never run dry
			v := lut[int(u*(lutN-1))]

			// Ease out towards the far corner, but never all the way to nothing.
			t := 1 - math.Hypot(fx-srcX, fy-srcY)/maxDist
			t = math.Min(1, math.Max(0, t))
			falloff := 0.25 + 0.75*(t*t*(3-2*t))

			delta := o.strength * (v - 0.35) * falloff

			i := canvas.PixOffset(x, y)
			for c := range 3 {
				p := float64(canvas.Pix[i+c]) / 255
				// Screen the light, multiply the shade: light can't blow out a
				// white window and shade can't crush a black backdrop.
				if delta > 0 {
					p += delta * (1 - p)
				} else {
					p += delta * p
				}
				canvas.Pix[i+c] = clamp8(p)
			}
		}
	}
}
