package main

import (
	"image"
	"image/color"
	"math"
	"sort"
)

// A mesh gradient blends several colour blobs placed across the canvas, rather
// than sweeping between stops along one axis. It reads as richer and less flat.

type meshPoint struct {
	x, y float64 // position in unit canvas coordinates
	c    color.RGBA
}

type meshPreset struct {
	name   string
	points []meshPoint
	spread float64 // blob radius as a fraction of the diagonal
	isDark bool
}

func mp(x, y float64, hex string) meshPoint {
	c, err := parseHex(hex)
	if err != nil {
		panic(err)
	}
	return meshPoint{x, y, c}
}

var meshPresets = []meshPreset{
	{name: "nebula", spread: 0.32, isDark: true, points: []meshPoint{
		mp(0.05, 0.10, "#4C1D95"), mp(0.95, 0.05, "#DB2777"),
		mp(0.15, 0.95, "#1E3A8A"), mp(0.90, 0.90, "#7C3AED"),
		mp(0.50, 0.45, "#9333EA"),
	}},
	{name: "sunrise", spread: 0.32, points: []meshPoint{
		mp(0.10, 0.15, "#FDE68A"), mp(0.90, 0.10, "#FB7185"),
		mp(0.05, 0.90, "#FDBA74"), mp(0.95, 0.95, "#C026D3"),
		mp(0.50, 0.50, "#FB923C"),
	}},
	{name: "lagoon", spread: 0.32, points: []meshPoint{
		mp(0.05, 0.05, "#67E8F9"), mp(0.95, 0.15, "#2563EB"),
		mp(0.10, 0.95, "#14B8A6"), mp(0.90, 0.90, "#0F766E"),
		mp(0.55, 0.45, "#38BDF8"),
	}},
	{name: "iris", spread: 0.32, points: []meshPoint{
		mp(0.08, 0.20, "#6366F1"), mp(0.92, 0.08, "#22D3EE"),
		mp(0.15, 0.92, "#A855F7"), mp(0.88, 0.95, "#EC4899"),
		mp(0.50, 0.50, "#818CF8"),
	}},
	{name: "ember", spread: 0.32, isDark: true, points: []meshPoint{
		mp(0.10, 0.10, "#7F1D1D"), mp(0.90, 0.05, "#F97316"),
		mp(0.05, 0.95, "#450A0A"), mp(0.95, 0.90, "#DC2626"),
		mp(0.45, 0.55, "#B91C1C"),
	}},
	{name: "meadow", spread: 0.32, points: []meshPoint{
		mp(0.05, 0.10, "#BBF7D0"), mp(0.95, 0.05, "#5EEAD4"),
		mp(0.10, 0.95, "#A3E635"), mp(0.90, 0.95, "#0891B2"),
		mp(0.50, 0.50, "#34D399"),
	}},
}

func meshNames() []string {
	n := make([]string, 0, len(meshPresets))
	for _, p := range meshPresets {
		n = append(n, p.name)
	}
	sort.Strings(n)
	return n
}

func findMesh(name string) (meshPreset, bool) {
	for _, p := range meshPresets {
		if p.name == name {
			return p, true
		}
	}
	return meshPreset{}, false
}

// renderMesh blends the blobs with inverse-Gaussian weights, in linear light so
// overlapping colours mix the way light does rather than turning to mud.
func renderMesh(r image.Rectangle, p meshPreset) *image.RGBA {
	dst := image.NewRGBA(r)
	w, h := float64(r.Dx()), float64(r.Dy())
	diag := math.Hypot(w, h)
	sigma := p.spread * diag
	twoSigmaSq := 2 * sigma * sigma

	type lin struct{ r, g, b float64 }
	pts := make([]struct {
		x, y float64
		c    lin
	}, len(p.points))
	for i, q := range p.points {
		pts[i].x, pts[i].y = q.x*w, q.y*h
		pts[i].c = lin{toLinear(float64(q.c.R) / 255), toLinear(float64(q.c.G) / 255), toLinear(float64(q.c.B) / 255)}
	}

	for y := r.Min.Y; y < r.Max.Y; y++ {
		fy := float64(y - r.Min.Y)
		for x := r.Min.X; x < r.Max.X; x++ {
			fx := float64(x - r.Min.X)
			var sr, sg, sb, sw float64
			for _, q := range pts {
				dx, dy := fx-q.x, fy-q.y
				wt := math.Exp(-(dx*dx + dy*dy) / twoSigmaSq)
				sr += q.c.r * wt
				sg += q.c.g * wt
				sb += q.c.b * wt
				sw += wt
			}
			if sw == 0 {
				sw = 1
			}
			d := (bayer8[y&7][x&7]/64 - 0.5) / 255
			i := dst.PixOffset(x, y)
			dst.Pix[i+0] = clamp8(toSRGB(sr/sw) + d)
			dst.Pix[i+1] = clamp8(toSRGB(sg/sw) + d)
			dst.Pix[i+2] = clamp8(toSRGB(sb/sw) + d)
			dst.Pix[i+3] = 0xFF
		}
	}
	return dst
}
