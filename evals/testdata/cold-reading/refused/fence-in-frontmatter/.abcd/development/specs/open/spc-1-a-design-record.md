---
```
origin: ABCD-EVAL-REFUSED-FENCE-IN-FRONTMATTER
---

# A design record

The fence delimiter above sits INSIDE the frontmatter block. The fence mask
used to span the whole document, so that delimiter toggled the mask on and
every line after it inside the block was reported as fenced — and the excluded
key scan skips fenced lines. The key above therefore travelled, under a
manifest asserting that `origin` had been refused.
