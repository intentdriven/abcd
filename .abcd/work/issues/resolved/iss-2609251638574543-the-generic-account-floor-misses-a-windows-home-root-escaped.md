---
schema_version: 1
id: "iss-2609251638574543"
slug: "the-generic-account-floor-misses-a-windows-home-root-escaped"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
resolution: "endsWithPathFold (internal/adapter/scanner/identity.go) reads each backslash in an account-root prefix or the caller's home literal as a run of backslashes, bounded at maxSeparatorRun bytes per run and charged as it reads, so the single, doubled, quadrupled and deeper spellings of a Windows home are one account position; a run past the bound keeps the finding. TestLocalUsernameGenericAccountNameCaughtUnderAMultiplyEscapedWindowsRoot was watched failing on the quadrupled, eightfold and home-literal shapes before the change and passes after. The gap never reached a release: the generic floor it narrows came in with this lane."
impact: internal
resolved_by:
  commit: "a0c126b1"
---

The generic-account floor misses a Windows home root escaped more than once. accountRootPrefixes (internal/adapter/scanner/identity.go) lists the single and the doubled backslash spellings of \users\ and the home-literal clause of standsAsAccountName lists the home and its doubled spelling, so C:\\\\Users\\\\LOGIN\\\\Desktop, the shape a transcript line carries when a tool result is itself JSON text (go env -json, npm config ls --json, any --json output), raises no finding for a login on the generic list while the doubled spelling hard-fails. Before the generic floor the bare word was flagged, so the floor narrowed a hard_fail rule in the redactor input: capture and history redact nothing on such a line.

## Grounds

- pursued: a generic login under a Windows home root escaped at any depth up to maxSeparatorRun is reported as local_username; a line carrying C:\Users\LOGIN with its separators quadrupled or more that yields no local_username finding, or a meter fixture of escaped roots whose charge grows faster than the line, would show it wrong.
