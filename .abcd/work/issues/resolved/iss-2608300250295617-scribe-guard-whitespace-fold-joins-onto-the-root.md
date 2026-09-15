---
schema_version: 1
id: "iss-2608300250295617"
slug: "scribe-guard-whitespace-fold-joins-onto-the-root"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "itd-188 sixth-round security review, 2026-08-30"
found_at: "internal/core/lint/scribecontract_test.go (scribeSpacedSeparatorRe, scribeFold)"
resolution: "The whitespace collapse has to exist — internal / core / lint is one path written with spaces — but applied after a trailing separator it glues the next token onto the allowed root, so a shipped-tree path written after .abcd/work/issues/ became a segment under it and passed the prefix check. Neither view sees both spellings, so scribeFold now returns both the decoded-but-uncollapsed text and the collapsed text, the path scan runs over each, and the findings are unioned and de-duplicated: the joined view still catches the spaced path, the unjoined view catches the smuggled token as itself. Three bypass cases of class path pin it — a bare-prose join with a space, the same inside one code span, and a TAB variant — each watched red first, and each still red when the scan is cut back to the joined view alone. The conforming base passes and no existing case changed class."
impact: internal
---

The scribe guard's spaced-separator fold is a join, not only a fold: horizontal whitespace after an allowed root's trailing slash glues the next token onto it, so a shipped-tree path written in plain prose after .abcd/work/issues/ (space or TAB, bare or in one code span) becomes a segment under the allowed root and passes the prefix check; demonstrated against the real test. Scan both the folded and the decoded-but-uncollapsed views and union the findings.
