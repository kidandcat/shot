# shot

Turn a plain screenshot into a presentable image: gradient background, generous
padding, rounded corners, a soft drop shadow, and an optional macOS or browser
window frame. A local CLI replacement for the web screenshot beautifiers
(postspark, pika.style, screely).

Single static binary, no runtime dependencies — the only system tools it shells
out to are `screencapture` (for `--capture`) and `osascript` (for `--copy`).

## Install

```sh
go build -o shot . && cp shot ~/.local/bin/
```

## Usage

```sh
# Beautify an existing screenshot
shot ugly.png --bg sunset --chrome mac

# Capture a window, frame it, and put it on the clipboard ready to paste in Slack
shot --capture window --bg midnight --chrome mac-dark --title "Marketplace" --copy

# Custom gradient, 16:9 canvas for a slide or a product update post
shot in.png --bg "#0f2027,#203a43,#2c5364" --ratio 16:9 -o out.png

# Your own image as the backdrop
shot in.png --bg ~/Pictures/wallpaper.jpg --pad 12%
```

Defaults are tuned to look right without any flags: `shot in.png` gives you the
`slate` gradient, 7% padding, a 12px radius and a soft shadow.

### Flags

| Flag | Default | Notes |
|---|---|---|
| `-o`, `--out` | `<input>-shot.png` | `~/Desktop/shot-<timestamp>.png` for captures |
| `--bg` | `slate` | preset name, `#hex[,#hex...]` gradient, or an image path |
| `--bg-angle` | `135` | 0 = left→right, 90 = top→bottom |
| `--pad` | `7%` | px or % of the window's long side |
| `--radius` | `12px` | scaled up on retina captures |
| `--shadow` | `3.5%` | blur radius; `0` disables |
| `--shadow-opacity` | `0.32` | |
| `--shadow-offset` | `1.1%` | vertical |
| `--shadow-color` | `#0A0C14` | |
| `--chrome` | `none` | `mac`, `mac-dark`, `browser`, `browser-dark`, `auto` |
| `--title` | | title bar text (mac chrome) |
| `--url` | | address bar text (browser chrome) |
| `--ratio` | | e.g. `16:9` — pads the canvas out, never crops |
| `--scale` | `1` | scale the input before compositing |
| `--capture` | | `window`, `region` or `screen` |
| `--copy` | | also copy the result to the clipboard |
| `--open` | | open the result when done |
| `--list-bg` | | list background presets |

Flags may appear before or after the input file.

Presets: `aurora`, `candy`, `dawn`, `flare`, `forest`, `grape`, `graphite`,
`ink`, `midnight`, `mint`, `ocean`, `peach`, `rose`, `sky`, `slate`, `sunset`.

`--chrome auto` picks the light or dark title bar based on how dark the
background is.

## Notes

- Gradients interpolate in linear light and are dithered, so large canvases
  don't band.
- `--capture window` passes `-o` to `screencapture` to drop the OS window
  shadow; `shot` renders its own.
- Linux capture needs `grim`+`slurp` or ImageMagick's `import`; clipboard needs
  `wl-copy` or `xclip`.
