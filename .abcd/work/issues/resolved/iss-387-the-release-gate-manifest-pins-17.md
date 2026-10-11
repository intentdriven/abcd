---
schema_version: 1
id: "iss-387"
slug: "the-release-gate-manifest-pins-17"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: ".abcd/development/release-gate/manifest.json"
resolution: "pin the five unpinned shipped surface chapters (guard/ideate/identity/banlist/update) in the release-gate manifest; checkerCount 27 (22 briefDocs + 5 surfaces); README counts corrected"
impact: fix
---

the release-gate manifest pins 17 briefDocs while four shipped surface chapters (guard, ideate, identity, banlist) are outside the pin, so every full-tier crosscheck attests coverage it never ran
## Evidence

- `.abcd/development/release-gate/manifest.json:22-37` pins 17 `briefDocs`; `brief/04-surfaces/README.md` rows 17-20 (`/abcd:guard`, `/abcd:ideate`, `/abcd:identity`, `/abcd:banlist`) are Status: shipped with chapters `17-guard.md`..`20-banlist.md` on disk, none pinned. `brief-surface-crosscheck.js:52` reads only `input.briefDocs` — disk is never consulted.
- Proof the chapters were skipped, not passed: the newest full-tier receipt (`.abcd/work/reviews/dc4de340…/iss35-brief-surface-crosscheck.json`, `manifestHash` equal to the current file) contains findings only for the 17 pinned docs + 5 surfaces — zero from chapters 17-20, while `git ls-tree` shows all four present at the content commit. `release-gate/README.md:86-87` ("the 17-document brief-doc list", "22 = 17 brief docs + 5 surfaces", full tier = "the whole brief-doc list") makes `full` a false coverage claim riding on a signed attestation.
- Not a freeze: the manifest was edited between cuts (`manifestHash` changed across the v0.4.x/v0.5.0 receipts, consistent with the `16-audit.md` → `16-lint.md` rename), and the runbook has no refresh step — the pin simply was not maintained as surfaces shipped.
- Refuter verdict: CONFIRMED (minor, drift; top-priority record item). The pinned prompt's 10-of-19 command roster is a separate, known-but-unrecorded deferral (`DECISIONS.md:919`: `promptHash` pins prompt text "by an algorithm nothing in the tree computes or verifies, so it is left untouched") — the prompt text and `promptHash` are deliberately NOT edited by this round's fix.

## Resolution note

Pre-merge review caught that the first fix was itself N−1 of N: the shipped
`21-update.md` chapter (`/abcd:update`, itd-130, merged the same morning) was a
fifth unpinned chapter. The final fix pins all five (guard/ideate/identity/
banlist/update), giving 22 briefDocs + 5 surfaces = checkerCount 27.
