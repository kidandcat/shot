package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
)

func loadImage(path string) (image.Image, error) {
	var rd io.Reader
	if path == "-" {
		rd = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		rd = f
	}
	img, _, err := image.Decode(rd)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return img, nil
}

func saveImage(path string, img image.Image, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		flat := image.NewRGBA(img.Bounds())
		draw.Draw(flat, flat.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
		draw.Draw(flat, flat.Bounds(), img, img.Bounds().Min, draw.Over)
		return jpeg.Encode(f, flat, &jpeg.Options{Quality: quality})
	default:
		enc := png.Encoder{CompressionLevel: png.DefaultCompression}
		return enc.Encode(f, img)
	}
}

func toRGBA(src image.Image) *image.RGBA {
	if r, ok := src.(*image.RGBA); ok {
		return r
	}
	dst := image.NewRGBA(image.Rect(0, 0, src.Bounds().Dx(), src.Bounds().Dy()))
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	return dst
}

func resize(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

// roundedMask renders an anti-aliased rounded-rectangle coverage mask.
func roundedMask(w, h int, radius float64) *image.Alpha {
	m := image.NewAlpha(image.Rect(0, 0, w, h))
	radius = math.Min(radius, math.Min(float64(w), float64(h))/2)
	if radius <= 0 {
		for i := range m.Pix {
			m.Pix[i] = 0xFF
		}
		return m
	}
	corners := [4][2]float64{
		{radius, radius},
		{float64(w) - radius, radius},
		{radius, float64(h) - radius},
		{float64(w) - radius, float64(h) - radius},
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			cov := 1.0
			inCornerBox := (px < radius || px > float64(w)-radius) && (py < radius || py > float64(h)-radius)
			if inCornerBox {
				// Nearest corner centre; coverage falls off across one pixel at the arc.
				best := math.Inf(1)
				for _, c := range corners {
					if d := math.Hypot(px-c[0], py-c[1]); d < best {
						best = d
					}
				}
				cov = math.Min(1, math.Max(0, radius-best+0.5))
			}
			m.Pix[m.PixOffset(x, y)] = uint8(math.Round(cov * 255))
		}
	}
	return m
}

func applyMask(src image.Image, mask *image.Alpha) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.DrawMask(dst, dst.Bounds(), src, b.Min, mask, image.Point{}, draw.Over)
	return dst
}

// boxBlurAlpha approximates a Gaussian blur of an alpha plane with three box
// passes (Wells' approximation) — visually indistinguishable and O(n) in radius.
func boxBlurAlpha(src *image.Alpha, sigma float64) *image.Alpha {
	if sigma <= 0 {
		return src
	}
	sizes := boxSizesForGauss(sigma, 3)
	cur := src
	for _, s := range sizes {
		r := (s - 1) / 2
		if r < 1 {
			continue
		}
		cur = boxBlurPass(cur, r)
	}
	return cur
}

func boxSizesForGauss(sigma float64, n int) []int {
	wIdeal := math.Sqrt((12 * sigma * sigma / float64(n)) + 1)
	wl := int(math.Floor(wIdeal))
	if wl%2 == 0 {
		wl--
	}
	wu := wl + 2
	mIdeal := (12*sigma*sigma - float64(n*wl*wl) - float64(4*n*wl) - float64(3*n)) / float64(-4*wl-4)
	m := int(math.Round(mIdeal))
	sizes := make([]int, n)
	for i := 0; i < n; i++ {
		if i < m {
			sizes[i] = wl
		} else {
			sizes[i] = wu
		}
	}
	return sizes
}

func boxBlurPass(src *image.Alpha, r int) *image.Alpha {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	tmp := image.NewAlpha(b)
	dst := image.NewAlpha(b)

	// Horizontal, sliding-window sum with clamped edges.
	for y := 0; y < h; y++ {
		row := src.Pix[y*src.Stride : y*src.Stride+w]
		out := tmp.Pix[y*tmp.Stride : y*tmp.Stride+w]
		var sum int
		for i := -r; i <= r; i++ {
			sum += int(row[clampi(i, 0, w-1)])
		}
		n := 2*r + 1
		for x := 0; x < w; x++ {
			out[x] = uint8(sum / n)
			sum += int(row[clampi(x+r+1, 0, w-1)]) - int(row[clampi(x-r, 0, w-1)])
		}
	}
	for x := 0; x < w; x++ {
		var sum int
		at := func(y int) int { return int(tmp.Pix[clampi(y, 0, h-1)*tmp.Stride+x]) }
		for i := -r; i <= r; i++ {
			sum += at(i)
		}
		n := 2*r + 1
		for y := 0; y < h; y++ {
			dst.Pix[y*dst.Stride+x] = uint8(sum / n)
			sum += at(y+r+1) - at(y-r)
		}
	}
	return dst
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// fillCircle draws an anti-aliased filled circle.
func fillCircle(dst *image.RGBA, cx, cy, r float64, c color.RGBA) {
	x0, y0 := int(cx-r-1), int(cy-r-1)
	x1, y1 := int(cx+r+2), int(cy+r+2)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if !(image.Pt(x, y).In(dst.Bounds())) {
				continue
			}
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			a := math.Min(1, math.Max(0, r-d+0.5))
			if a <= 0 {
				continue
			}
			blendPixel(dst, x, y, c, a)
		}
	}
}

// fillRoundRect draws an anti-aliased filled rounded rect at an offset.
func fillRoundRect(dst *image.RGBA, r image.Rectangle, radius float64, c color.RGBA) {
	m := roundedMask(r.Dx(), r.Dy(), radius)
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			a := float64(m.Pix[m.PixOffset(x, y)]) / 255
			if a <= 0 {
				continue
			}
			blendPixel(dst, r.Min.X+x, r.Min.Y+y, c, a)
		}
	}
}

func blendPixel(dst *image.RGBA, x, y int, c color.RGBA, a float64) {
	if !image.Pt(x, y).In(dst.Bounds()) {
		return
	}
	a *= float64(c.A) / 255
	i := dst.PixOffset(x, y)
	dst.Pix[i+0] = uint8(float64(c.R)*a + float64(dst.Pix[i+0])*(1-a))
	dst.Pix[i+1] = uint8(float64(c.G)*a + float64(dst.Pix[i+1])*(1-a))
	dst.Pix[i+2] = uint8(float64(c.B)*a + float64(dst.Pix[i+2])*(1-a))
	dst.Pix[i+3] = uint8(255*a + float64(dst.Pix[i+3])*(1-a))
}

// dropShadow renders a blurred, offset silhouette of a rounded rect.
func dropShadow(w, h int, radius, sigma, offsetY float64, opacity float64, c color.RGBA) (*image.RGBA, image.Point) {
	pad := int(math.Ceil(sigma*3)) + int(math.Abs(offsetY)) + 2
	sw, sh := w+2*pad, h+2*pad
	silhouette := image.NewAlpha(image.Rect(0, 0, sw, sh))
	m := roundedMask(w, h, radius)
	for y := 0; y < h; y++ {
		copy(silhouette.Pix[(y+pad)*silhouette.Stride+pad:(y+pad)*silhouette.Stride+pad+w],
			m.Pix[y*m.Stride:y*m.Stride+w])
	}
	blurred := boxBlurAlpha(silhouette, sigma)

	out := image.NewRGBA(image.Rect(0, 0, sw, sh))
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			a := float64(blurred.Pix[blurred.PixOffset(x, y)]) / 255 * opacity
			if a <= 0 {
				continue
			}
			i := out.PixOffset(x, y)
			out.Pix[i+0] = uint8(float64(c.R) * a)
			out.Pix[i+1] = uint8(float64(c.G) * a)
			out.Pix[i+2] = uint8(float64(c.B) * a)
			out.Pix[i+3] = uint8(a * 255)
		}
	}
	return out, image.Pt(-pad, -pad+int(offsetY))
}
