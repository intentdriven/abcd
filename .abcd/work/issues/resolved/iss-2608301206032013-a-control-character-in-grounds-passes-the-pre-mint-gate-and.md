---
schema_version: 1
id: "iss-2608301206032013"
slug: "a-control-character-in-grounds-passes-the-pre-mint-gate-and"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-2-security"
found_at: "internal/core/capture/promote.go"
resolution: "ValidateText refuses any rune below 0x20, so a control character is caught at the argument boundary all three grounds writers cross rather than by yamlScalar after the draft has been minted"
impact: fix
resolved_by:
  intent: "itd-179"
---

a control character in --grounds passes the pre-mint gate and faults inside the locked stamp leaving an orphan draft

Found by the round-2 adversarial security review of build/itd-179.

Go's RE2 `\s` does not include the vertical tab, so `grounds.Fold` keeps
U+000B and `grounds.New` accepts it. `requireGrounds` at
`internal/core/capture/promote.go:79` is called before anything is minted --
its own comment says a refusal raised later "would leave an orphan draft
behind" -- and it returns clean. The mint then runs, and `setScalarField` ->
`yamlScalar` rejects the control character under the ledger lock, after the
draft exists. Reproduced end to end: the error names the orphaned draft and the
repair verb.

The orphan-with-a-repair-verb contract is pre-existing, so the severity is
minor; what is new is that caller-supplied text can reach it. On main every
value reaching the stamp (`promoted_to`, `impact`) was validated before the
mint.

Remedy: `grounds.ValidateText` refuses what `yamlScalar` refuses (any rune
below 0x20), so the class is caught at the argument boundary for all three
routes rather than at one of them.

## Grounds

- pursued: we expect a pre-mint gate to be worth having only if it refuses everything the post-mint serialiser refuses, and a vertical tab reaching the mint and orphaning a draft is what showed the gap
