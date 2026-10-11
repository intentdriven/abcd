---
schema_version: 1
id: "iss-2609251823559111"
slug: "the-capture-ledger-s-os-root-escape-is"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "both ledger race tests assert errors.Is(err, ErrPathUnsafe) and a direct test feeds mapEscape a real os.Root escape, so a reworded Go message fails the capture tests"
impact: internal
resolved_by:
  commit: "20da9778c"
---

The capture ledger's os.Root escape is classified by matching the string 'path escapes from parent' (internal/core/capture/ledgerroot.go:59-64), and nothing pins the match: ledgerroot_test.go:62 and :97 assert only err != nil, so a Go release that rewords the message would silently degrade ErrPathUnsafe to a generic error (review-capture 1). Assert errors.Is(err, ErrPathUnsafe) in both race tests.

## Grounds

- pursued: a Go release that rewords the os.Root escape text turns TestMapEscapeClassifiesTheRealOSRootEscape and the write-race test RED; a mutant of the match string leaving the capture tests green would show it wrong
