# shot

Turn a plain screenshot into a presentable image: gradient background, generous
padding, rounded corners, a soft drop shadow, and an optional macOS or browser
window frame. A local CLI replacement for the web screenshot beautifiers
(postspark, pika.style, screely).

Single static binary, no runtime dependencies — the only system tools it shells
out to are `screencapture` (for `--capture`) and `osascript` (for `--copy`).

## Install

```sh
git clone https://github.com/kidandcat/shot && cd shot && ./install.sh
```

That builds the binary into `~/.local/bin` and symlinks the [Claude Code
skill](skills/shot/SKILL.md) into `~/.claude/skills`, so Claude knows how to
drive it. Pass `--no-skill` for the binary alone, or use Go directly:

```sh
go install github.com/kidandcat/shot@latest
```

## Usage

```sh
# Screenshot a website headless — no real browser UI in the image
shot https://fecha.app --copy
shot https://fecha.app --full-page --dark -o out.png

# Beautify an existing screenshot
shot ugly.png --bg sunset --chrome mac

# Capture a window, frame it, and put it on the clipboard ready to paste in Slack
shot --capture window --bg midnight --chrome mac-dark --title "Marketplace" --copy

# Custom gradient, 16:9 canvas for a slide or a product update post
shot in.png --bg "#0f2027,#203a43,#2c5364" --ratio 16:9 -o out.png

# Diagonal blind shadows across the whole image
shot in.png --bg slate --rays

# Your own image as the backdrop
shot in.png --bg ~/Pictures/wallpaper.jpg --pad 12%
```

Defaults are tuned to look right without any flags: `shot in.png` gives you the
`sunrise` mesh gradient, soft diagonal light rays, 7% padding, a 12px radius and
a soft shadow.

### Flags

| Flag | Default | Notes |
|---|---|---|
| `-o`, `--out` | `<input>-shot.png` | `~/Desktop/shot-<timestamp>.png` for captures |
| `--bg` | `sunrise` | preset name, `#hex[,#hex...]` gradient, or an image path |
| `--bg-angle` | `135` | 0 = left→right, 90 = top→bottom |
| `--pad` | `7%` | px or % of the window's long side |
| `--radius` | `12px` | scaled up on retina captures |
| `--shadow` | `3.5%` | blur radius; `0` disables |
| `--shadow-opacity` | `0.32` | |
| `--shadow-offset` | `1.1%` | vertical |
| `--shadow-color` | `#0A0C14` | |
| `--no-rays` | | disable the light rays (on by default) |
| `--rays-strength` | `0.07` | |
| `--rays-angle` | `35` | |
| `--rays-width` | `2.5` | >1 = fewer, wider bands |
| `--rays-under` | | confine the rays to the backdrop, sparing the window |
| `--chrome` | `none` | `mac`, `mac-dark`, `browser`, `browser-dark`, `auto` |
| `--title` | | title bar text (mac chrome) |
| `--url` | | address bar text (browser chrome) |
| `--ratio` | | e.g. `16:9` — pads the canvas out, never crops |
| `--scale` | `1` | scale the input before compositing |
| `--capture` | | `window`, `region`, `screen`, or `rect:X,Y,W,H` (non-interactive) |
| `--width` / `--height` | `1440` / `900` | page viewport for URL renders |
| `--dsf` | `2` | device pixel ratio for URL renders |
| `--full-page` | | capture the whole page, not just the viewport |
| `--wait` | `1200` | ms to let the page settle before rendering |
| `--dark` | | render the page with `prefers-color-scheme: dark` |
| `--copy` | | also copy the result to the clipboard |
| `--open` | | open the result when done |
| `--list-bg` | | list background presets |

Flags may appear before or after the input file.

Mesh presets (multi-colour blobs, richer than a two-stop sweep): `nebula`,
`sunrise`, `lagoon`, `iris`, `ember`, `meadow`.

Linear gradient presets: `aurora`, `candy`, `dawn`, `flare`, `forest`, `grape`, `graphite`,
`ink`, `midnight`, `mint`, `ocean`, `peach`, `rose`, `sky`, `slate`, `sunset`.

`--chrome auto` picks the light or dark title bar based on how dark the
background is.

## Websites

Passing a URL renders it in a headless Chrome over CDP (via `chromedp`) and
frames the result. Nothing of the real browser's UI is in the image — the
address bar you see is drawn by `shot`, so it stays clean whatever your Chrome
looks like. `--chrome browser` and `--url <the url>` are implied; override
either, or pass `--chrome none` for a bare page.

Requires Google Chrome or Chromium on the machine. Output defaults to
`<host>-shot.png`.

## Notes

- Gradients interpolate in linear light and are dithered, so large canvases
  don't band.
- `--capture window` passes `-o` to `screencapture` to drop the OS window
  shadow; `shot` renders its own.
- Linux capture needs `grim`+`slurp` or ImageMagick's `import`; clipboard needs
  `wl-copy` or `xclip`.
