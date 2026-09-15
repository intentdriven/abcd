<!-- a leading comment, and nothing else, before the block -->
---
origin: ABCD-EVAL-REFUSED-DISPLACED-BLOCK
---

# Framing

## Construal

The delimited block above does not open at line 0, so this binary reads it as
prose and never scans it for excluded keys — while a reader of the bundle, and
every YAML-aware reader, reads it as frontmatter. The key inside it travelled
under a manifest asserting that `origin` had been refused.
