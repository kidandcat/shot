---
name: shot
description: Turn a plain screenshot — or a live website — into a presentable image: mesh-gradient backdrop, soft light rays, rounded corners, drop shadow, optional macOS/browser window frame. Use whenever the user asks to make a screenshot look good, "beautify"/"prettify" a screenshot, screenshot a website or landing page, prepare an image for a product update, a blog post, a slide, Slack or social media, or asks for a postspark / pika.style / screely / CleanShot-style image. Also use to capture a window or region and frame it in one step.
---

# shot

`shot` is a local CLI that does what the screenshot-beautifier web apps do, without one. Give it a PNG or a URL, get back a framed image.

Check it is installed with `shot --version`. If it is missing, build it from the repo (`go build -o shot . && cp shot ~/.local/bin/`).

## The short version

The defaults are already tuned. Do not add flags unless the user asks for something specific.

```sh
shot screenshot.png                 # writes screenshot-shot.png
shot screenshot.png --copy --open   # also to the clipboard, and open it
shot https://fecha.app --copy       # renders the page headless, writes fecha-app-shot.png
```

That gives a `sunrise` mesh gradient, soft diagonal light rays, 7% padding, a 12px radius and a soft drop shadow.

## Websites

Pass a URL and `shot` renders it in a headless Chrome over CDP. **No part of the real browser's UI reaches the image** — no tabs, no bookmarks, no extensions. The address bar you see in the output is drawn by `shot`, so it looks the same on every machine. This is the right way to screenshot a page; do not drive the user's visible browser and capture the screen.

```sh
shot https://fecha.app                        # viewport render, drawn browser frame
shot https://fecha.app --full-page            # the whole page, however long
shot https://fecha.app --dark                 # prefers-color-scheme: dark
shot https://fecha.app --width 390 --height 844   # a phone viewport
shot https://fecha.app --chrome none          # bare page, no drawn frame
shot https://fecha.app --wait 3000            # let a heavy page settle longer
```

`--chrome browser` and `--url <the url>` are implied for URLs; override them if you want a different address bar text, or `--chrome none` for no frame. `--dsf 3` (default) renders well above retina density, so the image survives being downscaled by Slack, X or a blog. `--dsf 4` goes further at a real cost in time and file size. Needs Chrome or Chromium installed; a bad URL fails with the network error, it does not produce a blank image.

## Flags worth knowing

| Want | Flag |
|---|---|
| A different backdrop | `--bg <preset\|#hex[,#hex...]\|image.jpg>` (`--list-bg` to see presets) |
| No backdrop, for placing on any page | `--bg transparent` (PNG only; JPEG flattens to white; no light rays) |
| No light rays | `--no-rays` |
| Stronger / weaker rays | `--rays-strength 0.12` (default `0.07`) |
| Wider / narrower ray bands | `--rays-width 4` (default `2.5`) |
| Rays only on the backdrop | `--rays-under` |
| A macOS window frame | `--chrome mac` (or `mac-dark`, `auto`) with `--title "..."` |
| A browser frame | `--chrome browser` with `--url "..."` |
| Fit a slide | `--ratio 16:9` |
| More / less breathing room | `--pad 12%` |
| Output path | `-o out.png` (`.jpg` also works) |

Backgrounds come in two families: **mesh** (`nebula`, `sunrise`, `lagoon`, `iris`, `ember`, `meadow`) blend several colour blobs and look livelier; **linear gradients** (`slate`, `dawn`, `midnight`, `ocean`, …) sweep between stops. Flags may go before or after the input file. `-` reads stdin.

## Choosing a look

- **Do not add `--chrome` to a screenshot that already shows a window.** A capture of a Slack or browser window already has its own title bar, and you would end up with two. (URLs are the exception: a headless render has no frame, so `shot` adds one for you.)
- Rays are subtle on saturated backdrops and obvious on pale ones. If the user says the effect is invisible, raise `--rays-strength`; if it looks busy, lower it.
- `--chrome auto` picks a light or dark title bar based on how dark the backdrop is.

## Capturing

```sh
shot --capture window --copy          # click a window
shot --capture region -o out.png      # drag a rectangle
shot --capture rect:0,30,1920,1050    # no interaction; explicit rectangle
```

`rect:X,Y,W,H` is the one to use when you already know the window's geometry and do not want to make the user click. On macOS you can read it with:

```sh
osascript -e 'tell application "System Events" to tell process "Slack" to get {position, size} of front window'
```

### If capture fails with "could not create image from display"

That is macOS denying Screen Recording, not a bug in `shot`. The permission is granted to the *responsible process*, which for an agent shelling out is usually the terminal app — or nothing at all when the process is detached from one.

Ask the user which app already holds the permission (**System Settings → Privacy & Security → Screen Recording**), then run `shot` from that app. For a terminal with the permission already granted:

```sh
osascript -e 'tell application "iTerm" to create window with default profile command "/path/to/script.sh"'
```

No program can grant itself this permission. Do not retry the same failing command.

### Capture hygiene

- **Bring the target window to the front and give it a beat to redraw**, or the capture grabs whatever is on top instead. Verify afterwards rather than assuming — a quick check that the frame looks like what you expected (say, mostly dark for a dark-mode app) catches the wrong window before the user sees it.
- **Park the mouse cursor off the content first.** Slack, GitHub and many apps paint a hover toolbar under the pointer, and it will land in the shot. `cliclick m:960,8` puts it in the menu bar.
- Passing `--capture window` uses `screencapture -o`, which drops the OS window shadow, because `shot` renders its own.

## Processing, not capturing

If the user already has a screenshot, just process it. Never re-run a capture to redo the styling — and never feed an already-processed `*-shot.png` back into `shot`, or you will composite a backdrop on top of a backdrop. Keep the original around if you plan to try several looks:

```sh
for s in 0.04 0.07 0.12; do shot raw.png --rays-strength $s -o "try-$s.png"; done
```
