---
id: dcn-2608260000000006
slug: bug-hunt-round-9-bughunt-a-baseline
date: 2026-08-26
---
2026-08-26 — Bug-hunt round 9 (bughunt-a): baseline green after unshallowing
the environment's clone. Five parallel hunters returned 21 candidates across
the four dimensions; per-finding adversarial refutation confirmed 5
substantive and 4 fixable nitpicks and refuted or deferred 12. Fixed and
resolved with provenance: the memory-store guarded-read sweep (every reader
outside ingest followed a committed symlink unbounded — five reproduced
hangs), the issue-resolution gate's swallowed git probes (a dubious-ownership
refusal read as an empty ledger, exit 0), lint-config severity validation
with strict decoding (off-enum severities counted toward no exit code),
the docs-lint renderer's unsanitised config-derived fields, runBounded's
whole-buffer trim corrupting the first NUL-list entry, privacy-hygiene's
silent skip of unreadable tracked files, the terminology page's corpus
"ships" overclaim, the intake S4 container pin below the module toolchain,
and the site command description's missing check-write disclosure. Recorded
open: the scanIntentTree ENOENT-parting alignment (all claimed triggers
proved closed upstream). Refuted with prior art: the spc-28 planned/drafts
sweep (iss-94 convention), the verification-matrix capture-promote row
(iss-2608231346137587 embargo), the attribution fence guard (iss-270
wontfix; zero verdict flips across all historical PR bodies), isHexSHA
40-hex (prior round's refutation stands), frontmatterOpen's comment latch,
the skew-meta precedence claim, guard check's exit-2-on-disabled (specified),
the prepare-this-repo bucket gloss, the MADR label, the itd-27 pronoun
candidate (a cited real person is outside the persona rule), and the
record-lint job-name mismatch (mirror is correct by construction).
nitpicks-only: no.
