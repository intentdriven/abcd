---
schema_version: 1
id: "iss-2610121521482863"
slug: "fsutil-foldpath-the-canonical-fold-for"
severity: "minor"
category: "observation"
source: "impl-review"
found_during: "security review round 2 of fix/quota-deadline-and-folded-branch, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/paths.go"
remedy: "move FoldPath to full case folding plus NFC (golang.org/x/text/cases, already a dependency), sweep its callers for the direction each one fails safe, and let the loop's refFold call it"
---

fsutil.FoldPath, the canonical fold for names a case-folding filesystem resolves, lower-cases with Go's simple mapping and then applies NFC, but APFS folds with full Unicode case folding over normalised text. It treats straße and strasse, final and medial sigma, the micro sign and mu, and the fi ligature and fi as one name, where FoldPath keeps them apart, and FoldPath equates a dotted capital I with a plain i, where APFS keeps them apart. Every FoldPath caller (the payload destination check, the pack overlap check, embark's claimed key) can therefore read two spellings of one directory as distinct. The loop's branch check now folds fully through its own refFold; the shared helper does not.
