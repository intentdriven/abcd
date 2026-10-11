---
schema_version: 1
id: "iss-296"
slug: "the-hand-maintained-adr-index-table-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: ".abcd/development/decisions/adrs/README.md"
resolution: "Added adr-43/adr-44 rows to the ADR index; corrected the ruleset README count seven -> eight."
impact: fix
---

The hand-maintained ADR index table (.abcd/development/decisions/adrs/README.md) ends at adr-42 and omits committed records adr-43 and adr-44; the record-lint index_drift rule registers four indexes but not the ADR index, so this largest hand index is ungated. Same file: .abcd/work/rulesets/README.md says 'seven required status checks' where main-protection.json now lists eight (external-review added)
## Evidence

- `.abcd/development/decisions/adrs/README.md` — the hand-maintained index table's last row is
  `adr-42`; committed records `0043-inbound-equals-outbound-and-the-org-role.md`
  (`adr-43`, accepted) and `0044-remote-mutation-and-caller-identity.md`
  (`adr-44`, proposed) have no rows. The README states appending the row is a manual edit.
- `.abcd/record-lint.json` — the `index_drift` rule registers four indexes (`commands`,
  `planned-seams`, `research-children`, `later-phase-intents`); the ADR index is not among
  them, so the drift is ungated. `docs-lint.json` roots exclude `.abcd/development`.
- `.abcd/work/rulesets/README.md` says "the seven required status checks"; 
  `.abcd/work/rulesets/main-protection.json` lists eight contexts (external-review added):
  attribution, check (macos-latest), check (ubuntu-latest), external-review, gitleaks,
  record-lint, smoke, zizmor.

## Adversarial review

CONFIRMED (substantive/nitpick) by two independent refuters: the ADR-index omission is real,
committed on main, and structurally ungated (not covered by iss-38's four indexes); the
ruleset-README miscount is real and inside the file whose purpose is to mirror the live
ruleset. Fix: append the two ADR rows and correct "seven" to "eight".
