# Slides

Slides in your terminal — personal fork of [maaslalani/slides](https://github.com/maaslalani/slides)
with real LaTeX rendering, real images (including animated GIFs) with
captions, numbered bibliography/citations, and progressive reveal, built
for academic and scientific presentations.

<p align="center">
  <img src="./assets/slides-1.gif?raw=true" alt="Slides Presentation" />
</p>

### What's different from upstream

* **LaTeX math** — `$...$` inline math renders as Unicode text inline with
  the surrounding prose; `$$...$$` block math renders as a real image via
  [tectonic](https://tectonic-typesetting.github.io/) + Ghostscript.
* **Real images** — `![alt](src "caption")` shows the actual image (not
  just alt text) via the Kitty Graphics Protocol, with the caption
  rendered below it, including animated GIFs playing back for real. Falls
  back to plain alt text on terminals without Kitty graphics support.
* **Bibliography & citations** — `[@key]` citations are numbered by order
  of first appearance and resolved against a `.bib` file, with a
  References slide generated automatically at the end. See
  `bibliography` under [Configuration](#configuration).
* **Progressive reveal** — `<!-- pause -->` splits a slide into steps
  revealed one keypress at a time.
* **Plots from executed code** — any image file a `ctrl+e` code block
  saves (a Julia/Python/R plot, ...) is shown below its output; Julia
  specifically gets ANSI-colored UnicodePlots and a headless GR/Plots.jl
  backend for free.
* Layout/margin/sizing tuned for presenting on a projector (see
  `examples/` for a real worked example — not checked in, but built the
  same way you would build your own).

Everything else — navigation, search, code execution, SSH — works exactly
like upstream `slides`, documented below.

### Installation

This fork isn't published to any package manager. Build it from source:

```
git clone <this repo>
cd slides
go install
```

Or, with the Nix flake in this repo (also brings in `tectonic` and
`ghostscript` for LaTeX rendering, and `julia-bin` for the Julia code
execution extras):

```
nix build   # ./result/bin/slides
# or, for a dev shell with go/gopls/tectonic/ghostscript on PATH:
nix develop
```

### Usage
Create a simple markdown file that contains your slides:

````markdown
# Welcome to Slides
A terminal based presentation tool

---

## Everything is markdown
In fact, this entire presentation is a markdown file.

---

## Everything happens in your terminal
Create slides and present them without ever leaving your terminal.

---

## Code execution
```go
package main

import "fmt"

func main() {
  fmt.Println("Execute code directly inside the slides")
}
```

You can execute code inside your slides by pressing `<C-e>`,
the output of your command will be displayed at the end of the current slide.

---

## Pre-process slides

You can add a code block with three tildes (`~`) and write a command to run *before* displaying
the slides, the text inside the code block will be passed as `stdin` to the command
and the code block will be replaced with the `stdout` of the command.

```
~~~graph-easy --as=boxart
[ A ] - to -> [ B ]
~~~
```

The above will be pre-processed to look like:

┌───┐  to   ┌───┐
│ A │ ────> │ B │
└───┘       └───┘

For security reasons, you must pass a file that has execution permissions
for the slides to be pre-processed. You can use `chmod` to add these permissions.

```bash
chmod +x file.md
```

````

Checkout the [example slides](./examples).

Then, to present, run:
```
slides presentation.md
```

If given a file name, `slides` will automatically look for changes in the file and update the presentation live.

`slides` also accepts input through `stdin`:
```
curl http://example.com/slides.md | slides
```

Go to the first slide with the following key sequence:
* <kbd>g</kbd> <kbd>g</kbd>

Go to the next slide with any of the following key sequences:
* <kbd>space</kbd>
* <kbd>right</kbd>
* <kbd>down</kbd>
* <kbd>enter</kbd>
* <kbd>n</kbd>
* <kbd>j</kbd>
* <kbd>l</kbd>
* <kbd>Page Down</kbd>
* number + any of the above (go forward n slides)

Go to the previous slide with any of the following key sequences:
* <kbd>left</kbd>
* <kbd>up</kbd>
* <kbd>p</kbd>
* <kbd>h</kbd>
* <kbd>k</kbd>
* <kbd>N</kbd>
* <kbd>Page Up</kbd>
* number + any of the above (go back n slides)

Go to a specific slide with the following key sequence:

* number + <kbd>G</kbd>

Go to the last slide with the following key:

* <kbd>G</kbd>

### Search

To quickly jump to the right slide, you can use the search function.

Press <kbd>/</kbd>, enter your search term and press <kbd>Enter</kbd>  
(*The search term is interpreted as a regular expression. The `/i` flag causes case-insensitivity.*).

Press <kbd>ctrl+n</kbd> after a search to go to the next search result.

### Code Execution

If slides finds a code block on the current slides it can execute the code block and display the result as virtual text
on the screen.

Press <kbd>ctrl+e</kbd> on a slide with a code block to execute it and display the result.

Execution runs in the background, so a slow first run (Julia compiling a
plotting library, for example) doesn't freeze the presentation — a spinner
shows while it's working. If the code saves an image file (a plot,
typically), it's shown below the text output, the same way a regular
image would be.

Julia gets three extras, all automatic, no configuration needed:

* [UnicodePlots.jl](https://github.com/JuliaPlots/UnicodePlots.jl) draws
  straight into the code output, in color, sized to fit the slide.
* [GR.jl](https://github.com/jheinen/GR.jl) / [Plots.jl](https://docs.juliaplots.org)
  with the GR backend never tries to open a window — `GKSwstype` is set
  for you.
* [CairoMakie](https://docs.makie.org/stable/) (or anything else that
  just `save()`s a PNG) works out of the box. GLMakie and other
  GPU/window-backed backends aren't supported — there's no display to
  hand them.

### Progressive reveal

Split a slide into steps, revealed one at a time, with an HTML comment on
its own line:

```
- always visible
<!-- pause -->
- appears on the next keypress
```

The same keys that page forward/back (space, `j`/`k`, arrows, ...) reveal
or retract one step first, before moving to the next slide. Jumping
(`3j`, `5G`, `gg`) always lands on the target slide fully revealed.

### Pre-processing

You can add a code block with three tildes (`~`) and write a command to run
*before* displaying the slides, the text inside the code block will be passed
as `stdin` to the command and the code block will be replaced with the `stdout`
of the command. Wrap the pre-processed block in three backticks to keep
proper formatting and new lines.

````
```
~~~graph-easy --as=boxart
[ A ] - to -> [ B ]
~~~
```
````

The above will be pre-processed to look like:

```
┌───┐  to   ┌───┐
│ A │ ────> │ B │
└───┘       └───┘
```

For security reasons, you must pass a file that has execution permissions
for the slides to be pre-processed. You can use `chmod` to add these permissions.

```bash
chmod +x file.md
```

### LaTeX math

Write ordinary LaTeX math in your markdown:

```
Inline: $\alpha^2 + \beta \leq \gamma$, rendered as Unicode text.

Block:
$$\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}$$
```

Inline math (`$...$`) is approximated as Unicode text so it flows with
the surrounding paragraph. Block math (`$$...$$`) is compiled with
tectonic and rasterized with Ghostscript into a real image, shown the
same way as a regular image. If graphics aren't supported by your
terminal, block math falls back to the same Unicode approximation as
inline math.

### Images

```
![A caption for the image](./path/to/image.png "A caption for the image")
```

Shown as a real image via the Kitty Graphics Protocol (with the caption
rendered below it) on terminals that support it; falls back to plain alt
text everywhere else. A `.gif` with more than one frame plays back for
real, looping, entirely on the terminal's side.

### Bibliography & citations

Point `bibliography` (see [Configuration](#configuration)) at a `.bib`
file, then cite entries anywhere in your slides:

```
This result matches prior work [@smith2020].
```

Citations are numbered by order of first appearance across the whole
presentation; the same key cited again reuses its number. A `References`
slide listing every cited entry (IEEE-ish numeric style) is appended
automatically. Keys not found in the `.bib` file are left as literal
`[@key]` text.

### Configuration

`slides` allows you to customize your presentation's look and feel with metadata at the top of your `slides.md`.

> This section is entirely optional, `slides` will use sensible defaults if this section or any field in the section is omitted.

```yaml
---
theme: ./path/to/theme.json
author: Gopher
date: MMMM dd, YYYY
paging: Slide %d / %d
bibliography: ./path/to/refs.bib
---
```

* `theme`: Path to `json` file containing a [glamour
  theme](https://github.com/charmbracelet/glamour/tree/master/styles), can also
  be a link to a remote `json` file which slides will fetch before presenting.
* `author`: A `string` to display on the bottom-left corner of the presentation
  view. Defaults to the OS current user's full name. Can be empty to hide the author.
* `date`: A `string` that is used to format today's date in the `YYYY-MM-DD` format. If the date is not a valid
  format, the string will be displayed. Defaults to `YYYY-MM-DD`.
* `paging`: A `string` that contains 0 or more `%d` directives. The first `%d`
  will be replaced with the current slide number and the second `%d` will be
  replaced with the total slides count. Defaults to `Slide %d / %d`.
  You will need to surround the paging value with quotes if it starts with `%`.
* `bibliography`: Path to a `.bib` file, relative to the slides file. Enables
  `[@key]` citations and an auto-generated References slide. Omit to leave
  the feature off entirely.

#### Date format

Given the date _January 02, 2006_:

| Value  | Translates to |
|--------|---------------|
| `YYYY` | 2006          |
| `YY`   | 06            |
| `MMMM` | January       |
| `MMM`  | Jan           |
| `MM`   | 01            |
| `mm`   | 1             |
| `DD`   | 02            |
| `dd`   | 2             |

### SSH

Slides is accessible over `ssh` if hosted on a machine through the `slides
serve [file]` command.

On a machine, run:

```
slides serve [file]
```

Then, on another machine (or same machine), `ssh` into the port specified by
the `slides serve [file]` command:
```
ssh 127.0.0.1 -p 53531
```

You will be able to access the presentation hosted over SSH! You can use this
to present with `slides` from a computer that doesn't have `slides` installed,
but does have `ssh`. Or, let your viewers have access to the slides on their
own computer without needing to download `slides` and the presentation file.

### Credits

This is a personal fork of [maaslalani/slides](https://github.com/maaslalani/slides),
which was itself heavily inspired by [`lookatme`](https://github.com/d0c-s4vage/lookatme).

### Development
See the [development documentation](./docs/development)
