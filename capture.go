package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// validateRect checks an X,Y,W,H rectangle before it reaches screencapture -R.
func validateRect(rect string) error {
	parts := strings.Split(rect, ",")
	if len(parts) != 4 {
		return fmt.Errorf("--capture rect: expected rect:X,Y,W,H, got %q", rect)
	}
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return fmt.Errorf("--capture rect: %q is not a number", p)
		}
		if i >= 2 && v <= 0 {
			return fmt.Errorf("--capture rect: width and height must be > 0")
		}
	}
	return nil
}

// capture grabs a screenshot into a temp PNG and returns its path.
// mode is one of: window, region, screen, or rect:X,Y,W,H (non-interactive).
func capture(mode string) (string, error) {
	tmp, err := os.CreateTemp("", "shot-capture-*.png")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	tmp.Close()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		// -o drops the OS window shadow; we render our own.
		switch mode {
		case "window":
			cmd = exec.Command("screencapture", "-i", "-W", "-o", "-x", path)
		case "region":
			cmd = exec.Command("screencapture", "-i", "-s", "-x", path)
		case "screen":
			cmd = exec.Command("screencapture", "-m", "-x", path)
		default:
			rect, ok := strings.CutPrefix(mode, "rect:")
			if !ok {
				os.Remove(path)
				return "", fmt.Errorf("unknown capture mode %q (window|region|screen|rect:X,Y,W,H)", mode)
			}
			if err := validateRect(rect); err != nil {
				os.Remove(path)
				return "", err
			}
			cmd = exec.Command("screencapture", "-R"+rect, "-x", path)
		}
	case "linux":
		cmd, err = linuxCapture(mode, path)
		if err != nil {
			os.Remove(path)
			return "", err
		}
	default:
		os.Remove(path)
		return "", fmt.Errorf("--capture is not supported on %s", runtime.GOOS)
	}

	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("capture failed: %w", err)
	}
	// A cancelled interactive capture exits 0 without writing a file.
	if st, err := os.Stat(path); err != nil || st.Size() == 0 {
		os.Remove(path)
		return "", fmt.Errorf("capture cancelled")
	}
	return path, nil
}

func linuxCapture(mode, path string) (*exec.Cmd, error) {
	if _, err := exec.LookPath("grim"); err == nil {
		if mode == "region" {
			return exec.Command("sh", "-c", fmt.Sprintf("grim -g \"$(slurp)\" %q", path)), nil
		}
		return exec.Command("grim", path), nil
	}
	if _, err := exec.LookPath("import"); err == nil {
		if mode == "screen" {
			return exec.Command("import", "-window", "root", path), nil
		}
		return exec.Command("import", path), nil
	}
	return nil, fmt.Errorf("no capture backend found (install grim+slurp or imagemagick)")
}

// copyToClipboard puts the produced image on the system clipboard.
func copyToClipboard(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`set the clipboard to (read (POSIX file %q) as «class PNGf»)`, abs)
		return exec.Command("osascript", "-e", script).Run()
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			f, err := os.Open(abs)
			if err != nil {
				return err
			}
			defer f.Close()
			cmd := exec.Command("wl-copy", "--type", "image/png")
			cmd.Stdin = f
			return cmd.Run()
		}
		if _, err := exec.LookPath("xclip"); err == nil {
			return exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-i", abs).Run()
		}
		return fmt.Errorf("no clipboard backend found (install wl-clipboard or xclip)")
	default:
		return fmt.Errorf("--copy is not supported on %s", runtime.GOOS)
	}
}

func openFile(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Run()
	case "linux":
		return exec.Command("xdg-open", path).Run()
	default:
		return fmt.Errorf("--open is not supported on %s", runtime.GOOS)
	}
}
