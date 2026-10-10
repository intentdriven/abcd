---
schema_version: 1
id: "iss-364"
slug: "capture-validates-related-specs-against"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: "internal/core/capture/validate.go"
resolution: "Switched the related_specs validator from the retired reFnID (fn-N) to reSpcID (spc-N), removed the now-orphaned reFnID, and corrected issues/README.md; test updated to spc-12 (watched fail then pass)."
impact: fix
---

capture validates related_specs against the retired fn-N namespace (reFnID) instead of the live spc-N; adr-11 renamed fn to spec, fn- is a record-lint blocker, and 0 of 340 records populate the field, so it is unusable for its documented purpose
## Evidence

- `.abcd/work/issues/README.md:51` — `related_specs` documented as "list of `fn-N` ids".
- `internal/core/capture/validate.go:132` — `{"related_specs", reFnID, "fn-N"}`; `reFnID = ^fn-[0-9]+$` (`capture.go:240`). Nothing accepts `spc-N` for this field.
- adr-11 renamed fn→spec; live store is `spc-N` (`internal/core/lint/schema.go:50`). `.abcd/record-lint.json` bans `\bfn-` (blocker `flow-next-spec-prefix`), escaped here only because record-lint `roots` = `[.abcd/development]`.
- `reSpcID` already exists and is used (`workflow.go:222`, `--spec`/BySpec). Tests pin the retired form: `workflow_test.go:110,117` set `RelatedSpecs: []string{"fn-12"}`.
- 0 of 340 issue records populate `related_specs` — the field is unusable for its documented purpose.

## Adversarial verdict

CONFIRMED — code fix substantive (low impact), doc fix nitpick (must bundle with code). Refuter corrected two false sub-claims in the original hunter note (reSpcID is used, not dead; tests do pin fn-N) — the core drift survives both. Fix: `validate.go:132` `reFnID`→`reSpcID`, `"fn-N"`→`"spc-N"`; README:51 `fn-N`→`spc-N`; update `workflow_test.go:110,117` to `spc-12` (watched-fail: spc-12 rejected before, accepted after).
