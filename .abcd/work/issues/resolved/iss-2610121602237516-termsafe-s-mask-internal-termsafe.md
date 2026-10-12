---
schema_version: 1
id: "iss-2610121602237516"
slug: "termsafe-s-mask-internal-termsafe"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "security review of fix/sec-build-next-series, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/termsafe/termsafe.go"
remedy: "Add the tag characters (U+E0000-E007F), U+FFF9-FFFB, U+180E and U+034F to termsafe's masked set (and SanitizeBlock's), update the package's contract comment, and add a test per class; check the callers' tests still pass, since Sanitize-equality checks then refuse these runes too."
resolution: "termsafe's mask now covers the Unicode tag characters (U+E0000-E007F), the interlinear annotation marks (U+FFF9-FFFB), U+180E and U+034F, through Sanitize, SanitizeBlock, IsHidden and EncodeHiddenRunes alike; the package contract names them, and a test per class pins the mask."
impact: fix
resolved_by:
  commit: "8baebececbad593108e4c612ab1dd87f50ae477f"
---

termsafe's mask (internal/termsafe/termsafe.go, isZeroWidth and the bidi set) leaves several invisible rune classes unmasked: the Unicode tag characters U+E0000 to U+E007F, the interlinear annotation marks U+FFF9 to U+FFFB, U+180E (Mongolian vowel separator) and U+034F (combining grapheme joiner). They render invisibly, so text carrying them passes Sanitize (and every Sanitize-equality check built on it) unchanged and can hide content on a terminal or in a transcript; tag characters can also carry hidden instructions to an agent reading the output. Pre-existing across every Sanitize caller. Found by the security reviewer of fix/sec-build-next-series.

Security scope (abcd-fixes-what-it-adds): the route exists only through abcd's own masking helper and the surfaces that rely on it, so abcd fixes it; widening the mask changes nothing about how a repository works.
