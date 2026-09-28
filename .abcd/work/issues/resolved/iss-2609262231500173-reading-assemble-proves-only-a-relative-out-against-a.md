---
schema_version: 1
id: "iss-2609262231500173"
slug: "reading-assemble-proves-only-a-relative-out-against-a"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainFS"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/assemble.go"
resolution: "writeArtefacts proves every spelling of --out with gitutil.ProveOperandDir before creating anything; the relative in-repository case keeps EnsureRealDirAll"
impact: fix
resolved_by:
  commit: "6412b80d8"
---

reading assemble proves only a relative --out against a symlinked ancestor: writeArtefacts treats every ABSOLUTE --out as outside the repository (inRepo := !filepath.IsAbs(outDir) && ValidRelPath(rel)) and hands it to os.MkdirAll as given, so --out naming the checkout's own local tier absolutely, or climbing out and back in (../<repo>/...), is followed through a committed symlink at any level and the assembled input and manifest land at the link's target. Absolute is not outside; the sibling verb history reconstruct proves every spelling with gitutil.ProveOperandDir.

## Grounds

- pursued: an absolute --out and a ../<repo>/ --out below a committed local-tier link are both refused with nothing written at the link's target, while an absolute directory outside every checkout is still written; TestAssembleRefusesAnOutSpelledIntoTheCheckoutThroughALink failing on either spelling, or on the outside control, would show it wrong
