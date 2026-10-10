---
schema_version: 1
id: "iss-2608231025198888"
slug: "abcd-capture-writes-to-the-ledger"
severity: "major"
category: "security"
source: "user-observation"
found_during: "record-review"
found_at: "internal/core/capture/workflow.go"
details: "The redaction scanner was wired into launch (dryrun, ship), repolint (rule_privacy) and history (Capture) only. abcd capture rendered free text straight to the committed issue ledger with no detector between, so an absolute home path naming the operator's account reached a committed record with record-lint, docs-lint and lint-reviews all green. history.Capture refuses fail-closed on exactly this span class (home_path_self) while the ledger write path did not look."
suggested_fix: "Route both ledger write paths through the same detector: the capture render and the resolve/wontfix transition render. Redact and report rather than refuse, because refusing to record a finding loses the finding, and report the count so the caller knows the text was altered."
related_issues: ["iss-2608230847432286", "iss-2608230957104179"]
resolution: "capture redacts free-text inputs through the scanner BEFORE rendering (redactCaptureInputs) on both ledger write paths, the capture render and the resolve/wontfix transition, and reports the altered-span count. It REDACTS AND REPORTS and never refuses, unlike history.Capture: the asymmetry is deliberate (redact.go), because refusing to record a finding loses the finding itself"
impact: fix
resolved_by:
  commit: "2a4f4f2348ac02fa67758c807fc558968e5c8708"
---

abcd capture writes to the ledger without running the redaction scanner

The scanner was reachable from three places and none of them was the ledger:

```
internal/core/launch/dryrun.go, ship.go
internal/core/repolint/rule_privacy.go
internal/core/history/store.go, history.go
```

`abcd capture` rendered its frontmatter and body and wrote them. Nothing looked
at the bytes.

## How it surfaced

A record filed on 2026-08-23 carried an absolute home path naming the
operator's account, in a committed file, on a pushed branch. All three lint
gates passed. It was caught by a human reading the pull request, not by any
check.

The asymmetry is the sharp part. `history.Capture` treats this exact span class
as blocking: it scans, redacts, re-scans, and refuses the write if a
`home_path_self` survives. The transcript store — machine-produced bulk that is
gitignored and never leaves the machine — is fail-closed on home paths, while
the issue ledger — human-authored, committed, pushed, and published — had no
check at all. The more exposed artefact had the weaker guard.

## Why this is redact-and-report, not fail-closed

`history` refuses because a transcript is bulk: losing one costs a session, and
a session is reproducible. Refusing a capture loses the finding, and a ledger
that rejects writes stops being written to — which is the failure
`adversarial-review-scales-with-blast-radius` names when it insists capture stay
frictionless.

So both ledger write paths redact and report a count. Silence was not an option
either: redaction edits what the caller filed, and only they can judge whether
the redacted record still says what they meant, so the count reaches the
surface (loud-staging).

A degraded scanner — an unparseable per-repo `pii.json` — redacts with the
bundled defaults and returns a reason rather than refusing, because a broken
config file must not block every capture in the repo. That is a deliberate
divergence from `history`, which refuses on the same condition.

## Shape

The detector existed, measured the right property, and ran over a subject set
that excluded the ledger. That is the same narrowed-subject-set shape recorded
against iss-2608230847432286, in its third medium: a Go slice there, a prose
enumeration in the concurrency rule, and here a wiring list — the set of call
sites someone remembered to thread the scanner through.
