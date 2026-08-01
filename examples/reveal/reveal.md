---
author: Gopher
paging: Slide %d / %d
---

# Progressive reveal

Split a slide into steps with an HTML comment on its own line:

```
<!-- pause -->
```

Press the same keys you'd use to advance/go back — space, `j`, `k`, arrows —
to reveal or retract one step at a time. Jumping (`3j`, `5G`, `gg`) always
skips straight to the target slide, fully revealed.

---

# One point at a time

- First point, always visible
<!-- pause -->
- Second point, appears on the next keypress
<!-- pause -->
- Third point, appears on the keypress after that

---

# Works with anything on the slide

An equation can wait for its own step:

Rearranging Newton's second law:
<!-- pause -->
$$F = ma \implies a = \frac{F}{m}$$
