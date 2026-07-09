package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

// Website capture: drive a headless Chrome over CDP and grab the page itself.
// Nothing of the real browser's UI ends up in the image — any window frame in
// the output is the one shot draws.

type webOptions struct {
	width    int
	height   int
	scale    float64 // device pixel ratio; 2 gives a retina-sharp render
	fullPage bool
	wait     time.Duration
	dark     bool
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// displayURL is what goes in the drawn address bar: no scheme, no trailing slash.
func displayURL(raw string) string {
	s := strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	return strings.TrimSuffix(s, "/")
}

// urlOutput names the file after the host, e.g. https://fecha.app/x → fecha-app-shot.png
func urlOutput(raw string) string {
	host := displayURL(raw)
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host = host[:i]
	}
	host = strings.TrimPrefix(host, "www.")
	host = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-':
			return r
		default:
			return '-'
		}
	}, host)
	if host == "" {
		host = "page"
	}
	return host + "-shot.png"
}

// captureURL renders a page to a temp PNG and returns its path.
func captureURL(url string, o webOptions) (string, error) {
	if o.width <= 0 || o.height <= 0 {
		return "", fmt.Errorf("--width and --height must be > 0")
	}
	if o.scale <= 0 {
		o.scale = 2
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 90*time.Second)
	defer cancelTimeout()

	tasks := chromedp.Tasks{
		chromedp.EmulateViewport(int64(o.width), int64(o.height), chromedp.EmulateScale(o.scale)),
	}
	if o.dark {
		tasks = append(tasks, emulation.SetEmulatedMedia().
			WithFeatures([]*emulation.MediaFeature{{Name: "prefers-color-scheme", Value: "dark"}}))
	}
	tasks = append(tasks, chromedp.Navigate(url))
	// Give late-loading fonts, images and entry animations a chance to settle.
	tasks = append(tasks, chromedp.Sleep(o.wait))

	var buf []byte
	if o.fullPage {
		tasks = append(tasks, chromedp.FullScreenshot(&buf, 100))
	} else {
		tasks = append(tasks, chromedp.CaptureScreenshot(&buf))
	}

	if err := chromedp.Run(ctx, tasks); err != nil {
		if strings.Contains(err.Error(), "exec: ") || strings.Contains(err.Error(), "executable file not found") {
			return "", fmt.Errorf("could not launch a browser: install Google Chrome or Chromium (%w)", err)
		}
		return "", fmt.Errorf("render %s: %w", url, err)
	}
	if len(buf) == 0 {
		return "", fmt.Errorf("render %s: empty screenshot", url)
	}

	f, err := os.CreateTemp("", "shot-web-*.png")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(buf); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}
