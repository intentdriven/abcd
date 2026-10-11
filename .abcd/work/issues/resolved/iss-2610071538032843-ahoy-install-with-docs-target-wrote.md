---
schema_version: 1
id: "iss-2610071538032843"
slug: "ahoy-install-with-docs-target-wrote"
severity: "minor"
category: "bug"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610071228214583 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd ahoy install --answers, --docs-target"
remedy: "none (filed automatically)"
resolution: "a value flag that would change a saved setting now puts the config-change approval even with no config gap; a decline drops the flag, saves nothing and names it; --yes or approve.config-change applies it"
impact: fix
---

ahoy install with --docs-target wrote config and re-rendered AGENTS.md although every asked approval was answered later

Repository with docs.target saved as both (install refused until changed). Run: ahoy install --docs-target agents_md --answers FILE, where FILE answers approve.safe-autocreate and approve.oracle-routing with later. The run asked only those two approvals (no config-change approval was asked, though detection listed the docs-target gap as category config-change), then saved docs.target as agents_md and re-rendered the AGENTS.md block (98 to 104 lines).

The value was the person's answer, so the outcome was wanted, but the page says later declines and writes nothing, and the run read as a no-write probe. The earlier probe with no docs-target stopped before writing, which reinforced that expectation. Output was captured only partially; the interview record of the run lists just the two later answers.

Remedy the reporter proposes: Ask the config-change approval before applying a value flag's change, or say in the run's output that the flag applies without one; a run whose every asked approval was declined should report what it still wrote.

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610071228214583, a defect against abcd v0.13.1, surface abcd ahoy install --answers, --docs-target.

Evidence:

- rpt-2610071228214583 (the report, kept in the inbox)
- 571b1bf
