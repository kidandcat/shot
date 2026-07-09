package main

import (
	"flag"
	"image"
	"image/color"
	"reflect"
	"testing"
)

func TestParseHex(t *testing.T) {
	cases := map[string]color.RGBA{
		"#fff":      {0xFF, 0xFF, 0xFF, 0xFF},
		"#FF5F57":   {0xFF, 0x5F, 0x57, 0xFF},
		"0F2027":    {0x0F, 0x20, 0x27, 0xFF},
		"#00000080": {0, 0, 0, 0x80},
	}
	for in, want := range cases {
		got, err := parseHex(in)
		if err != nil || got != want {
			t.Errorf("parseHex(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "#12", "#gggggg", "nope"} {
		if _, err := parseHex(bad); err == nil {
			t.Errorf("parseHex(%q) should fail", bad)
		}
	}
}

func TestParseDim(t *testing.T) {
	if v, _ := parseDim("24", 1000); v != 24 {
		t.Errorf("px: got %v", v)
	}
	if v, _ := parseDim("7%", 1000); v != 70 {
		t.Errorf("pct: got %v", v)
	}
	if _, err := parseDim("wide", 100); err == nil {
		t.Error("want error for non-numeric")
	}
}

func TestExpandToRatio(t *testing.T) {
	// Never crops: the result contains the input box.
	w, h, err := expandToRatio(1000, 1000, "16:9")
	if err != nil || w < 1000 || h < 1000 {
		t.Fatalf("got %dx%d, %v", w, h, err)
	}
	if w != 1778 || h != 1000 {
		t.Errorf("got %dx%d, want 1778x1000", w, h)
	}
	if _, _, err := expandToRatio(10, 10, "16"); err == nil {
		t.Error("want error for malformed ratio")
	}
}

// permuteArgs is what lets flags appear after the positional input.
func TestPermuteArgs(t *testing.T) {
	newFS := func() *flag.FlagSet {
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.String("bg", "", "")
		fs.Bool("copy", false, "")
		return fs
	}
	got := permuteArgs(newFS(), []string{"in.png", "--bg", "sunset", "--copy"})
	want := []string{"--bg", "sunset", "--copy", "--", "in.png"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}

	fs := newFS()
	if err := fs.Parse(permuteArgs(newFS(), []string{"in.png", "--bg=ocean"})); err != nil {
		t.Fatal(err)
	}
	if fs.Arg(0) != "in.png" || fs.Lookup("bg").Value.String() != "ocean" {
		t.Errorf("inline value lost: arg=%q bg=%q", fs.Arg(0), fs.Lookup("bg").Value)
	}

	// "-" means stdin and must stay positional.
	if got := permuteArgs(newFS(), []string{"-", "--copy"}); !reflect.DeepEqual(got, []string{"--copy", "--", "-"}) {
		t.Errorf("stdin: got %q", got)
	}
}

func TestRoundedMaskCorners(t *testing.T) {
	m := roundedMask(100, 100, 20)
	if m.Pix[m.PixOffset(0, 0)] != 0 {
		t.Error("corner pixel should be transparent")
	}
	if m.Pix[m.PixOffset(50, 50)] != 0xFF {
		t.Error("centre pixel should be opaque")
	}
	// Radius is clamped to half the short side, so a huge radius is still valid.
	roundedMask(10, 4, 999)
}

func TestBoxBlurPreservesBounds(t *testing.T) {
	src := image.NewAlpha(image.Rect(0, 0, 32, 32))
	for i := range src.Pix {
		src.Pix[i] = 0xFF
	}
	out := boxBlurAlpha(src, 4)
	if out.Bounds() != src.Bounds() {
		t.Fatalf("bounds changed: %v", out.Bounds())
	}
	// A fully opaque plane stays opaque in the middle after blurring.
	if out.Pix[out.PixOffset(16, 16)] < 0xF0 {
		t.Errorf("centre washed out: %d", out.Pix[out.PixOffset(16, 16)])
	}
}

func TestComposeSizes(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 400, 300))
	base := options{bg: "slate", bgAngle: 135, pad: "10%", shadow: "0", chrome: "none", scale: 1, shadowOpacity: 0.3, shadowOffset: "0", shadowColor: "#000"}

	out, err := compose(src, base, false)
	if err != nil {
		t.Fatal(err)
	}
	// pad = 10% of the long side (400) = 40px per side.
	if got := out.Bounds().Dx(); got != 480 {
		t.Errorf("width = %d, want 480", got)
	}

	withChrome := base
	withChrome.chrome = "mac"
	out, err = compose(src, withChrome, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds().Dy() <= 380 {
		t.Errorf("chrome should add a title bar, got height %d", out.Bounds().Dy())
	}

	ratioed := base
	ratioed.ratio = "16:9"
	out, err = compose(src, ratioed, false)
	if err != nil {
		t.Fatal(err)
	}
	b := out.Bounds()
	if got := float64(b.Dx()) / float64(b.Dy()); got < 1.77 || got > 1.78 {
		t.Errorf("aspect = %v, want ~1.777", got)
	}
}

func TestComposeRejectsBadInput(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 10, 10))
	base := options{bg: "slate", pad: "0", shadow: "0", chrome: "none", scale: 1, shadowOffset: "0", shadowColor: "#000"}

	bad := base
	bad.bg = "not-a-preset"
	if _, err := compose(src, bad, false); err == nil {
		t.Error("want error for unknown background")
	}

	bad = base
	bad.chrome = "windows95"
	if _, err := compose(src, bad, false); err == nil {
		t.Error("want error for unknown chrome")
	}

	bad = base
	bad.scale = 0
	if _, err := compose(src, bad, false); err == nil {
		t.Error("want error for zero scale")
	}
}

func TestURLHelpers(t *testing.T) {
	for _, s := range []string{"https://a.com", "http://a.com"} {
		if !isURL(s) {
			t.Errorf("isURL(%q) = false", s)
		}
	}
	for _, s := range []string{"a.png", "-", "", "ftp://a.com", "httpsa.com"} {
		if isURL(s) {
			t.Errorf("isURL(%q) = true", s)
		}
	}

	if got := displayURL("https://fecha.app/"); got != "fecha.app" {
		t.Errorf("displayURL = %q", got)
	}
	if got := displayURL("http://a.com/x?y=1"); got != "a.com/x?y=1" {
		t.Errorf("displayURL = %q", got)
	}

	cases := map[string]string{
		"https://fecha.app/precios": "fecha-app-shot.png",
		"https://www.example.com":   "example-com-shot.png",
		"http://a.b.c/x?y=1#z":      "a-b-c-shot.png",
	}
	for in, want := range cases {
		if got := urlOutput(in); got != want {
			t.Errorf("urlOutput(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPresetsAreValid(t *testing.T) {
	for _, p := range presets {
		if len(p.stops) < 2 {
			t.Errorf("preset %q needs at least two stops", p.name)
		}
		if _, ok := findPreset(p.name); !ok {
			t.Errorf("preset %q not findable", p.name)
		}
	}
}
