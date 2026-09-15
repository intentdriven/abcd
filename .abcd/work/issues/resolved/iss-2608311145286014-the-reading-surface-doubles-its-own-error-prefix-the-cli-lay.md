---
schema_version: 1
id: "iss-2608311145286014"
slug: "the-reading-surface-doubles-its-own-error-prefix-the-cli-lay"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "itd-184 fidelity audit rcp-426034d44293"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/reading.go"
resolution: "The reading surface subtracts the core's tag instead of adding its own: readingRefusal() composes every reading sub-verb's refusal from the verb name plus the core message with the 'reading: ' prefix trimmed, and all four core-error paths in internal/surface/cli/reading.go go through it. The core is the canonical owner of the tag, because its strings are what the JSON envelope, the refusal records and the core's own tests read. Closes the half itd-185 left open on the assemble verb, and the bare verb it never covered."
impact: fix
resolved_by:
  commit: "7e6c86e98ed521fa1a406b285441775e3a8bae1a"
---

The reading surface doubles its own error prefix: the CLI layer prepends 'reading: ' and the locator's errors already carry it, so an operator sees 'abcd: reading: reading: agents/...' on a refusal. The path this appears on is the definition-refusal path spc-62 introduced as its own proof of wiring, so the first thing the new surface shows a human on failure is a stutter.

Half closed on 2026-08-31 by itd-185, and the record stays open for the half
that is not. The ingest verb's refusals no longer stutter: its surface trims the
prefix the core already carries. The assemble verb still prepends unconditionally
at internal/surface/cli/reading.go:49 while the core's errors carry the same
prefix, so `abcd reading assemble` still renders `abcd: reading: reading: ...` on
a refusal. It was left alone deliberately: it is a shipped verb's output rather
than the new one's, and changing it belongs with whoever is next in that file.

## Grounds

- pursued: one rule for the whole sub-tree removes the stutter everywhere rather than verb by verb; a surface test per sub-verb, text and --json alike, asserting exactly one tag on a CORE refusal would show it wrong — a row that still renders 'reading: reading: ', or a future sub-verb added without going through readingRefusal()
