# Slides

Slides in your terminal — personal fork of [maaslalani/slides](https://github.com/maaslalani/slides)
with real LaTeX, real images (including animated GIFs), bibliography/citations,
progressive reveal, and plots from executed code, built for academic and
scientific presentations.

<p align="center">
  <img src="./assets/slides-1.gif?raw=true" alt="Slides Presentation" />
</p>

### What's different from upstream

* **LaTeX math** — `$...$` inline math renders as Unicode text; `$$...$$`
  block math renders as a real image via [tectonic](https://tectonic-typesetting.github.io/) + Ghostscript.
* **Real images** — `![alt](src "caption")` shows the actual image via the
  Kitty Graphics Protocol, caption included, animated GIFs playing back
  for real. Falls back to plain alt text without Kitty graphics support.
* **Bibliography & citations** — `[@key]` citations are numbered by order
  of first appearance against a `.bib` file, with a References slide
  generated automatically. Enabled via `bibliography` in the frontmatter.
* **Progressive reveal** — `<!-- pause -->` splits a slide into steps
  revealed one keypress at a time.
* **Plots from executed code** — any image a `ctrl+e` code block saves is
  shown below its output. Julia specifically gets ANSI-colored
  UnicodePlots and a headless GR/Plots.jl/CairoMakie backend for free.
* Layout/margin/sizing tuned for presenting on a projector.

Everything else — navigation, search, pre-processing, SSH — works exactly
like upstream `slides`.

### Installation

This fork isn't published to any package manager. Build it from source:

```
git clone <this repo>
cd slides
go install
```

Or, with the Nix flake in this repo (also brings in `tectonic`,
`ghostscript`, and `julia-bin`):

```
nix build   # ./result/bin/slides
nix develop # dev shell with everything on PATH
```

### Usage

```
slides presentation.md
```

Space/`j`/`k`/arrows to move between slides, `/` to search, `ctrl+e` to
run a code block, `y` to yank one, `gg`/`G` to jump to the first/last
slide. `slides` also reads from `stdin` and reloads live on file changes.

Frontmatter (`theme`, `author`, `date`, `paging`, `bibliography`) is
optional — see [examples/basics/metadata.md](./examples/basics/metadata.md).

See [examples/](./examples) for every feature above in a working
presentation: LaTeX, images, GIFs, bibliography, reveal, Julia plots, and
everything inherited from upstream.

### Credits

This is a personal fork of [maaslalani/slides](https://github.com/maaslalani/slides),
which was itself heavily inspired by [`lookatme`](https://github.com/d0c-s4vage/lookatme).

### Development
See the [development documentation](./docs/development)
