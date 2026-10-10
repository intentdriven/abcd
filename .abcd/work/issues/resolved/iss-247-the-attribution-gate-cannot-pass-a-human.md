---
schema_version: 1
id: "iss-247"
slug: "the-attribution-gate-cannot-pass-a-human"
severity: "major"
category: "process"
source: "user-observation"
found_during: "adr-40-acceptance"
found_at: "scripts/check-attribution.sh"
resolution: "Assisted-by: None shipped in bb162f3 and is documented in CONTRIBUTING.md; the gate accepts a human-only disclosure"
impact: fix
---

The attribution gate cannot pass a human-only commit, so the maintainer cannot hand-edit a file through the web UI or any non-AI path. check-attribution.sh demands an 'Assisted-by:' trailer on every non-bot commit; the only exemption in .github/workflows/attribution.yml is for bot-authored pull requests. A manual one-line ADR status change (commit 5456f6022975, 'manual: Change status from proposed to accepted') failed the gate for having no trailer. The gate's own error text names the contradiction — 'If this artefact had no AI assistance, say so in the PR and this gate can be revisited; the convention is disclosure, and a human-only change has nothing to disclose. Today the gate asks for the trailer on every non-bot contribution.' Disclosure of AI assistance is the purpose, so demanding the disclosure where there was no assistance either forces a false trailer or blocks the commit. Today it blocks the maintainer; it will block the first outside human contribution the same way, and the DCO Signed-off-by deferred to the public flip does not resolve it. Needs a human-only path that is honest: an explicit no-assistance marker, or narrowing the gate to commits that claim assistance.