---
schema_version: 1
id: "iss-362"
slug: "abcd-banlist-json-omits-the-reach"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: "internal/core/banlist/private.go"
resolution: "Added a reach field to the banlist private JSON (PrivateReport), populated with PrivateReachNote unconditionally, mirroring the ahoy status board. Test extended to assert it."
impact: additive
---

abcd banlist --json omits the reach sentence its command file instructs an agent to relay verbatim; reach lives only in the text renderer, forcing the paraphrase the doc forbids, while abcd ahoy --json carries banlist.reach correctly
## Evidence

- `commands/banlist.md:26` gives `abcd banlist --json`; `:29` "Summarise the JSON"; `:34-35` "relay the `reach` sentence the render carries rather than paraphrasing it".
- `internal/core/banlist/private.go:55-86` — `PrivateReport` has no `reach` field; JSON keys are `path,present,keyed,entries,malformed_lines,inert_lines,not_ignored`.
- The reach sentence lives only in the text renderer: `internal/surface/cli/banlist.go:287` prints `banlist.PrivateReachNote` (constant `banlist.go:90-92`).
- Sibling `commands/ahoy.md:43-45` is correct — `abcd ahoy --json` carries `banlist.reach`.

## Adversarial verdict

CONFIRMED (substantive). The JSON-only section instructs relaying a sentence the JSON does not carry, forcing the paraphrase the paragraph forbids. Fix: add a `reach` field to the banlist private JSON populated with `PrivateReachNote`, mirroring ahoy; add a test.
