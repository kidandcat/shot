package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"sort"
	"strings"
)

// gradient is an ordered list of colour stops rendered as a linear gradient.
type gradient struct {
	name   string
	stops  []color.RGBA
	angle  float64 // degrees, 0 = left→right, 90 = top→bottom
	isDark bool    // hint for the default window chrome theme
}

var presets = []gradient{
	{name: "slate", stops: hexes("#E0EAFC", "#CFDEF3"), angle: 135},
	{name: "dawn", stops: hexes("#FDFBFB", "#E4E7EB"), angle: 135},
	{name: "sunset", stops: hexes("#FFB88C", "#DE6262"), angle: 135},
	{name: "peach", stops: hexes("#FFECD2", "#FCB69F"), angle: 135},
	{name: "ocean", stops: hexes("#2193B0", "#6DD5ED"), angle: 135},
	{name: "mint", stops: hexes("#A8EDEA", "#FED6E3"), angle: 135},
	{name: "grape", stops: hexes("#8E2DE2", "#4A00E0"), angle: 135, isDark: true},
	{name: "sky", stops: hexes("#56CCF2", "#2F80ED"), angle: 135},
	{name: "candy", stops: hexes("#FF9A9E", "#FAD0C4"), angle: 135},
	{name: "forest", stops: hexes("#134E5E", "#71B280"), angle: 135, isDark: true},
	{name: "midnight", stops: hexes("#0F2027", "#203A43", "#2C5364"), angle: 135, isDark: true},
	{name: "rose", stops: hexes("#F4C4F3", "#FC67FA"), angle: 135},
	{name: "flare", stops: hexes("#FF512F", "#F09819"), angle: 135},
	{name: "aurora", stops: hexes("#00C9FF", "#92FE9D"), angle: 135},
	{name: "graphite", stops: hexes("#232526", "#414345"), angle: 135, isDark: true},
	{name: "ink", stops: hexes("#1F1C2C", "#928DAB"), angle: 135, isDark: true},
}

func presetNames() []string {
	n := make([]string, 0, len(presets))
	for _, p := range presets {
		n = append(n, p.name)
	}
	sort.Strings(n)
	return n
}

func findPreset(name string) (gradient, bool) {
	for _, p := range presets {
		if p.name == name {
			return p, true
		}
	}
	return gradient{}, false
}

// parseHex accepts #RGB, #RRGGBB and #RRGGBBAA (leading # optional).
func parseHex(s string) (color.RGBA, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	switch len(s) {
	case 3:
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	case 6, 8:
	default:
		return color.RGBA{}, fmt.Errorf("bad hex colour %q", s)
	}
	c := color.RGBA{A: 0xFF}
	var v uint64
	for _, r := range s {
		var d uint64
		switch {
		case r >= '0' && r <= '9':
			d = uint64(r - '0')
		case r >= 'a' && r <= 'f':
			d = uint64(r-'a') + 10
		case r >= 'A' && r <= 'F':
			d = uint64(r-'A') + 10
		default:
			return color.RGBA{}, fmt.Errorf("bad hex colour %q", s)
		}
		v = v<<4 | d
	}
	if len(s) == 8 {
		c.R, c.G, c.B, c.A = uint8(v>>24), uint8(v>>16), uint8(v>>8), uint8(v)
	} else {
		c.R, c.G, c.B = uint8(v>>16), uint8(v>>8), uint8(v)
	}
	return c, nil
}

func hexes(list ...string) []color.RGBA {
	out := make([]color.RGBA, len(list))
	for i, h := range list {
		c, err := parseHex(h)
		if err != nil {
			panic(err)
		}
		out[i] = c
	}
	return out
}

// resolveBackground turns a --bg value into a renderer. The value is a preset
// name, a comma-separated list of hex colours (one = solid), or a path to an image.
func resolveBackground(spec string, angle float64, angleSet bool) (func(image.Rectangle) *image.RGBA, bool, error) {
	if p, ok := findPreset(spec); ok {
		a := p.angle
		if angleSet {
			a = angle
		}
		return func(r image.Rectangle) *image.RGBA { return renderGradient(r, p.stops, a) }, p.isDark, nil
	}

	if p, ok := findMesh(spec); ok {
		return func(r image.Rectangle) *image.RGBA { return renderMesh(r, p) }, p.isDark, nil
	}

	if strings.HasPrefix(spec, "#") || isHexList(spec) {
		parts := strings.Split(spec, ",")
		stops := make([]color.RGBA, 0, len(parts))
		for _, p := range parts {
			c, err := parseHex(p)
			if err != nil {
				return nil, false, err
			}
			stops = append(stops, c)
		}
		if len(stops) == 1 {
			stops = append(stops, stops[0])
		}
		return func(r image.Rectangle) *image.RGBA { return renderGradient(r, stops, angle) }, luma(stops[0]) < 0.55, nil
	}

	if _, err := os.Stat(spec); err == nil {
		src, err := loadImage(spec)
		if err != nil {
			return nil, false, err
		}
		return func(r image.Rectangle) *image.RGBA { return coverImage(src, r) }, averageLuma(src) < 0.55, nil
	}

	return nil, false, fmt.Errorf("unknown background %q (not a preset, hex colour or file)\ngradients: %s\nmesh: %s",
		spec, strings.Join(presetNames(), ", "), strings.Join(meshNames(), ", "))
}

func isHexList(s string) bool {
	for _, p := range strings.Split(s, ",") {
		if _, err := parseHex(p); err != nil {
			return false
		}
	}
	return s != ""
}

func luma(c color.RGBA) float64 {
	return (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
}

func averageLuma(img image.Image) float64 {
	b := img.Bounds()
	var sum float64
	var n float64
	step := max(1, b.Dx()/64)
	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			r, g, bl, _ := img.At(x, y).RGBA()
			sum += (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(bl)) / 65535
			n++
		}
	}
	if n == 0 {
		return 1
	}
	return sum / n
}

// bayer8 is an ordered dither matrix. Gradients across a large canvas quantise
// into visible bands at 8 bits per channel; a sub-LSB dither hides them.
var bayer8 = [8][8]float64{
	{0, 32, 8, 40, 2, 34, 10, 42},
	{48, 16, 56, 24, 50, 18, 58, 26},
	{12, 44, 4, 36, 14, 46, 6, 38},
	{60, 28, 52, 20, 62, 30, 54, 22},
	{3, 35, 11, 43, 1, 33, 9, 41},
	{51, 19, 59, 27, 49, 17, 57, 25},
	{15, 47, 7, 39, 13, 45, 5, 37},
	{63, 31, 55, 23, 61, 29, 53, 21},
}

func renderGradient(r image.Rectangle, stops []color.RGBA, angleDeg float64) *image.RGBA {
	dst := image.NewRGBA(r)
	w, h := float64(r.Dx()), float64(r.Dy())
	rad := angleDeg * math.Pi / 180
	dx, dy := math.Cos(rad), math.Sin(rad)

	// Project every corner onto the gradient axis so t spans exactly [0,1].
	var lo, hi float64
	for i, c := range [][2]float64{{0, 0}, {w, 0}, {0, h}, {w, h}} {
		p := c[0]*dx + c[1]*dy
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

	// Sampling in linear light per pixel means two pow() calls per channel;
	// a LUT along the gradient axis makes this ~40x cheaper at no visible cost.
	const lutN = 2048
	var lut [lutN][3]float64
	for i := range lut {
		cr, cg, cb := sampleStops(stops, float64(i)/(lutN-1))
		lut[i] = [3]float64{cr, cg, cb}
	}

	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			fx, fy := float64(x-r.Min.X), float64(y-r.Min.Y)
			t := (fx*dx + fy*dy - lo) / span
			idx := int(math.Min(1, math.Max(0, t)) * (lutN - 1))
			c := lut[idx]
			d := (bayer8[y&7][x&7]/64 - 0.5) / 255
			i := dst.PixOffset(x, y)
			dst.Pix[i+0] = clamp8(c[0] + d)
			dst.Pix[i+1] = clamp8(c[1] + d)
			dst.Pix[i+2] = clamp8(c[2] + d)
			dst.Pix[i+3] = 0xFF
		}
	}
	return dst
}

// sampleStops interpolates in linear light, which keeps two-stop gradients from
// going muddy through the midpoint the way naive sRGB blending does.
func sampleStops(stops []color.RGBA, t float64) (float64, float64, float64) {
	t = math.Min(1, math.Max(0, t))
	seg := t * float64(len(stops)-1)
	i := int(seg)
	if i >= len(stops)-1 {
		i = len(stops) - 2
	}
	f := seg - float64(i)
	a, b := stops[i], stops[i+1]
	mix := func(x, y uint8) float64 {
		return toSRGB(toLinear(float64(x)/255)*(1-f) + toLinear(float64(y)/255)*f)
	}
	return mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B)
}

func toLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func toSRGB(c float64) float64 {
	if c <= 0.0031308 {
		return c * 12.92
	}
	return 1.055*math.Pow(c, 1/2.4) - 0.055
}

func clamp8(v float64) uint8 {
	return uint8(math.Round(math.Min(1, math.Max(0, v)) * 255))
}

// coverImage scales src to fill r entirely (object-fit: cover) and centre-crops.
func coverImage(src image.Image, r image.Rectangle) *image.RGBA {
	sb := src.Bounds()
	scale := math.Max(float64(r.Dx())/float64(sb.Dx()), float64(r.Dy())/float64(sb.Dy()))
	w, h := int(math.Ceil(float64(sb.Dx())*scale)), int(math.Ceil(float64(sb.Dy())*scale))
	scaled := resize(src, w, h)
	dst := image.NewRGBA(r)
	off := image.Pt((w-r.Dx())/2, (h-r.Dy())/2)
	draw.Draw(dst, r, scaled, off, draw.Src)
	return dst
}
