---
author: Gopher
paging: Slide %d / %d
---

# Julia plots

Press `ctrl+e` on a Julia code block to run it. Text output (like
UnicodePlots) shows up right on the slide; if the code saves an image file
(`savefig`/`save`), it's shown below the output as a real image.

First run of each library can take a while (Julia precompiles it) --
that's expected, not a hang.

---

### UnicodePlots.jl

Draws straight to stdout, so it needs no image at all.

```julia
using UnicodePlots
println(lineplot(1:100, x -> sin(x / 5), title="sin(x / 5)"))
```

---

### Plots.jl + GR

`GKSwstype` is set automatically so this never tries to open a window.

```julia
using Plots
gr()
plot(1:100, x -> cos(x / 5), title="cos(x / 5)", legend=false)
savefig("plot.png")
```

---

### CairoMakie

```julia
using CairoMakie
fig = Figure()
ax = Axis(fig[1, 1], title = "x²")
lines!(ax, 1:100, x -> x^2)
save("fig.png", fig)
```
