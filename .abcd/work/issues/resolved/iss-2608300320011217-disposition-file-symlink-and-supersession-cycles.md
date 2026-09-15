---
schema_version: 1
id: "iss-2608300320011217"
slug: "disposition-file-symlink-and-supersession-cycles"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "itd-180 third-round security review, 2026-08-30"
found_at: "internal/core/capture/reading.go (readDispositions), internal/core/capture/promote.go, internal/core/lint/readingoutstanding.go, internal/core/issueschema/disposition.go"
resolution: "Disposition files are Lstat-refused in all three readers (capture's walk, promote's state read, lint's report) — a two-syscall check whose stat-then-read window closed only when iss-2608300349493306 moved them onto fsutil.ReadGuarded — a self-citing supersession is not well-formed so the record stands, and a non-empty record set with nothing standing is a supersession cycle the verb refuses and the board names."
impact: internal
---

The symlink refusal on the reading trees covers directories but not disposition files: readDispositions keeps a symlinked dsp-N.md, the standing computation, promote's state read and the board's exit-condition line then read a file outside the ledger, and the verb accepts a supersedes citation of the link; and a disposition whose supersedes_disposition names itself, or a two-record cycle, retires every standing answer so the item reads as undispositioned, the verb accepts an uncited fresh answer, and the board reports nothing. Lstat and IsRegular in all three readers; treat self-supersession as not well-formed and a non-empty record set with no standing answer as a ledger fault.
