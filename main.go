package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const version = "0.1.0"

type options struct {
	bg            string
	bgAngle       float64
	pad           string
	radius        string
	shadow        string
	shadowOpacity float64
	shadowOffset  string
	shadowColor   string
	noRays        bool
	raysStrength  float64
	raysAngle     float64
	raysWidth     float64
	raysUnder     bool
	chrome        string
	title         string
	url           string
	ratio         string
	scale         float64
	captureMode   string
	out           string
	quality       int
	copyClip      bool
	open          bool
	listBG        bool
	showVersion   bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "shot:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `shot — turn a plain screenshot into a presentable image

usage:
  shot [flags] <input.png|->      process an existing image ("-" reads stdin)
  shot --capture window [flags]   capture, then process

flags:
  -o, --out PATH       output file (default: <input>-shot.png, or ~/Desktop for captures)
      --bg SPEC        preset name, #hex[,#hex...] gradient, or an image path (default: sunrise)
      --bg-angle DEG   gradient angle, 0 = left→right, 90 = top→bottom
      --pad V          padding around the window, px or % of the long side (default: 7%)
      --radius V       window corner radius, px or % (default: 12px, scaled for retina)
      --shadow V       shadow blur radius, px or % (default: 3.5%); 0 disables
      --shadow-opacity F
      --shadow-offset V
      --shadow-color   #hex
      --no-rays        disable the window-blind light rays (on by default)
      --rays-strength F  default 0.04
      --rays-angle DEG   default 35
      --rays-width F     >1 = fewer, wider bands (default: 2.5)
      --rays-under     confine the rays to the backdrop, sparing the window
      --chrome KIND    none | mac | mac-dark | browser | browser-dark (default: none)
      --title TEXT     title bar text (mac chrome)
      --url TEXT       address bar text (browser chrome)
      --ratio W:H      pad the canvas out to an aspect ratio, e.g. 16:9
      --scale F        scale the input before compositing
      --capture MODE   window | region | screen
      --copy           also copy the result to the clipboard
      --open           open the result when done
      --list-bg        list background presets
      --version

examples:
  shot ugly.png --copy
  shot --capture window --bg midnight --chrome mac-dark --title "Marketplace" --copy
  shot in.png --bg "#0f2027,#2c5364" --ratio 16:9 -o out.png
`)
}

func run() error {
	var o options
	fs := flag.NewFlagSet("shot", flag.ContinueOnError)
	fs.Usage = usage
	fs.SetOutput(os.Stderr)

	str := func(p *string, val string, names ...string) {
		for _, n := range names {
			fs.StringVar(p, n, val, "")
		}
	}
	str(&o.bg, "sunrise", "bg")
	fs.Float64Var(&o.bgAngle, "bg-angle", 135, "")
	str(&o.pad, "7%", "pad", "padding")
	str(&o.radius, "", "radius")
	str(&o.shadow, "3.5%", "shadow")
	fs.Float64Var(&o.shadowOpacity, "shadow-opacity", 0.32, "")
	str(&o.shadowOffset, "1.1%", "shadow-offset")
	str(&o.shadowColor, "#0A0C14", "shadow-color")
	fs.BoolVar(&o.noRays, "no-rays", false, "")
	fs.Float64Var(&o.raysStrength, "rays-strength", 0.04, "")
	fs.Float64Var(&o.raysAngle, "rays-angle", 35, "")
	fs.Float64Var(&o.raysWidth, "rays-width", 2.5, "")
	fs.BoolVar(&o.raysUnder, "rays-under", false, "")
	str(&o.chrome, "none", "chrome")
	str(&o.title, "", "title")
	str(&o.url, "", "url")
	str(&o.ratio, "", "ratio")
	fs.Float64Var(&o.scale, "scale", 1, "")
	str(&o.captureMode, "", "capture")
	str(&o.out, "", "out", "o")
	fs.IntVar(&o.quality, "quality", 92, "")
	fs.BoolVar(&o.copyClip, "copy", false, "")
	fs.BoolVar(&o.open, "open", false, "")
	fs.BoolVar(&o.listBG, "list-bg", false, "")
	fs.BoolVar(&o.showVersion, "version", false, "")

	if err := fs.Parse(permuteArgs(fs, os.Args[1:])); err != nil {
		return err
	}
	if o.showVersion {
		fmt.Println("shot", version)
		return nil
	}
	if o.listBG {
		fmt.Println("gradients:", strings.Join(presetNames(), ", "))
		fmt.Println("mesh:     ", strings.Join(meshNames(), ", "))
		return nil
	}

	angleSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "bg-angle" {
			angleSet = true
		}
	})

	input := fs.Arg(0)
	var tempCapture string
	switch {
	case o.captureMode != "":
		p, err := capture(o.captureMode)
		if err != nil {
			return err
		}
		tempCapture, input = p, p
		defer os.Remove(tempCapture)
	case input == "":
		usage()
		return fmt.Errorf("no input (pass a file, \"-\", or --capture)")
	}

	src, err := loadImage(input)
	if err != nil {
		return err
	}

	out, err := compose(src, o, angleSet)
	if err != nil {
		return err
	}

	dst := o.out
	if dst == "" {
		dst = defaultOutput(input, tempCapture != "")
	}
	if err := saveImage(dst, out, o.quality); err != nil {
		return err
	}

	if o.copyClip {
		if err := copyToClipboard(dst); err != nil {
			return fmt.Errorf("copy to clipboard: %w", err)
		}
	}
	if o.open {
		if err := openFile(dst); err != nil {
			return err
		}
	}

	b := out.Bounds()
	fmt.Printf("%s (%d×%d)", dst, b.Dx(), b.Dy())
	if o.copyClip {
		fmt.Print(" · copied to clipboard")
	}
	fmt.Println()
	return nil
}

func compose(src image.Image, o options, angleSet bool) (image.Image, error) {
	if o.scale != 1 {
		if o.scale <= 0 {
			return nil, fmt.Errorf("--scale must be > 0")
		}
		b := src.Bounds()
		src = resize(src, int(float64(b.Dx())*o.scale), int(float64(b.Dy())*o.scale))
	}

	// Retina captures need proportionally bigger chrome, or the title bar reads
	// as a hairline next to the content.
	uiScale := math.Min(3, math.Max(1, float64(src.Bounds().Dx())/1400))

	bgFn, bgDark, err := resolveBackground(o.bg, o.bgAngle, angleSet)
	if err != nil {
		return nil, err
	}

	window := toRGBA(src)
	if o.chrome != "" && o.chrome != "none" {
		kind, dark, err := parseChrome(o.chrome, bgDark)
		if err != nil {
			return nil, err
		}
		window = addChrome(window, chromeStyle{kind: kind, dark: dark, title: o.title, url: o.url}, uiScale)
	}

	wb := window.Bounds()
	long := float64(max(wb.Dx(), wb.Dy()))

	radius := 12 * uiScale
	if o.radius != "" {
		if radius, err = parseDim(o.radius, long); err != nil {
			return nil, fmt.Errorf("--radius: %w", err)
		}
	}
	pad, err := parseDim(o.pad, long)
	if err != nil {
		return nil, fmt.Errorf("--pad: %w", err)
	}
	sigma, err := parseDim(o.shadow, long)
	if err != nil {
		return nil, fmt.Errorf("--shadow: %w", err)
	}
	offset, err := parseDim(o.shadowOffset, long)
	if err != nil {
		return nil, fmt.Errorf("--shadow-offset: %w", err)
	}
	shColor, err := parseHex(o.shadowColor)
	if err != nil {
		return nil, fmt.Errorf("--shadow-color: %w", err)
	}

	rounded := applyMask(window, roundedMask(wb.Dx(), wb.Dy(), radius))

	canvasW := wb.Dx() + 2*int(math.Round(pad))
	canvasH := wb.Dy() + 2*int(math.Round(pad))
	if o.ratio != "" {
		canvasW, canvasH, err = expandToRatio(canvasW, canvasH, o.ratio)
		if err != nil {
			return nil, err
		}
	}

	canvas := bgFn(image.Rect(0, 0, canvasW, canvasH))
	origin := image.Pt((canvasW-wb.Dx())/2, (canvasH-wb.Dy())/2)

	// Underlay rays light the backdrop only; the window is composited on top of
	// them, so there is no point shading the pixels it will cover.
	if !o.noRays && o.raysUnder {
		mask := roundedMask(wb.Dx(), wb.Dy(), radius)
		applyRays(canvas, o.raysOpts(), mask, origin)
	}

	if sigma > 0 && o.shadowOpacity > 0 {
		sh, shOff := dropShadow(wb.Dx(), wb.Dy(), radius, sigma, offset, o.shadowOpacity, shColor)
		at := origin.Add(shOff)
		draw.Draw(canvas, sh.Bounds().Add(at), sh, sh.Bounds().Min, draw.Over)
	}
	draw.Draw(canvas, rounded.Bounds().Add(origin), rounded, rounded.Bounds().Min, draw.Over)

	if !o.noRays && !o.raysUnder {
		applyRays(canvas, o.raysOpts(), nil, image.Point{})
	}
	return canvas, nil
}

func (o options) raysOpts() raysOptions {
	return raysOptions{strength: o.raysStrength, angle: o.raysAngle, width: o.raysWidth}
}

func parseChrome(spec string, bgDark bool) (kind string, dark bool, err error) {
	switch spec {
	case "mac":
		return "mac", false, nil
	case "mac-dark":
		return "mac", true, nil
	case "browser":
		return "browser", false, nil
	case "browser-dark":
		return "browser", true, nil
	case "auto":
		return "mac", bgDark, nil
	}
	return "", false, fmt.Errorf("--chrome: unknown kind %q (none|mac|mac-dark|browser|browser-dark|auto)", spec)
}

// permuteArgs moves positional arguments after the flags. Go's flag package
// stops parsing at the first non-flag, so `shot in.png --bg sunset` would
// otherwise drop every flag on the floor without complaining.
func permuteArgs(fs *flag.FlagSet, args []string) []string {
	isBool := func(name string) bool {
		f := fs.Lookup(name)
		if f == nil {
			return false
		}
		bf, ok := f.Value.(interface{ IsBoolFlag() bool })
		return ok && bf.IsBoolFlag()
	}

	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			positional = append(positional, args[i+1:]...)
			return append(flags, append([]string{"--"}, positional...)...)
		case a == "-" || !strings.HasPrefix(a, "-"):
			positional = append(positional, a)
		default:
			flags = append(flags, a)
			name, _, hasValue := strings.Cut(strings.TrimLeft(a, "-"), "=")
			if !hasValue && !isBool(name) && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		}
	}
	if len(positional) == 0 {
		return flags
	}
	return append(flags, append([]string{"--"}, positional...)...)
}

// parseDim reads "24" (pixels) or "7%" (fraction of ref).
func parseDim(s string, ref float64) (float64, error) {
	s = strings.TrimSpace(s)
	if pct, ok := strings.CutSuffix(s, "%"); ok {
		v, err := strconv.ParseFloat(strings.TrimSpace(pct), 64)
		if err != nil {
			return 0, err
		}
		return v / 100 * ref, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("expected a number or percentage, got %q", s)
	}
	return v, nil
}

// expandToRatio grows the canvas (never crops) until it matches W:H.
func expandToRatio(w, h int, ratio string) (int, int, error) {
	parts := strings.Split(ratio, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("--ratio: expected W:H, got %q", ratio)
	}
	rw, err1 := strconv.ParseFloat(parts[0], 64)
	rh, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil || rw <= 0 || rh <= 0 {
		return 0, 0, fmt.Errorf("--ratio: expected W:H, got %q", ratio)
	}
	if float64(w)/float64(h) < rw/rh {
		return int(math.Round(float64(h) * rw / rh)), h, nil
	}
	return w, int(math.Round(float64(w) * rh / rw)), nil
}

func defaultOutput(input string, fromCapture bool) string {
	stamp := time.Now().Format("20060102-150405")
	if fromCapture || input == "-" {
		if home, err := os.UserHomeDir(); err == nil {
			desktop := filepath.Join(home, "Desktop")
			if st, err := os.Stat(desktop); err == nil && st.IsDir() {
				return filepath.Join(desktop, "shot-"+stamp+".png")
			}
		}
		return "shot-" + stamp + ".png"
	}
	ext := filepath.Ext(input)
	return strings.TrimSuffix(input, ext) + "-shot.png"
}
