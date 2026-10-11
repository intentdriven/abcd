---
schema_version: 1
id: "iss-2610071538028804"
slug: "ahoy-reports-no-gaps-while-abcd-lint"
severity: "minor"
category: "inconsistency"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610071228213002 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd ahoy, abcd ahoy doctor, abcd lint (three-tier-layout, decision-durability)"
remedy: "none (filed automatically)"
resolution: "ahoy now detects and creates the committed tiers three-tier-layout requires, read from one shared list, and seeds .abcd/work/DECISIONS.md with a header"
impact: fix
---

ahoy reports no gaps while abcd lint errors on the missing shared work tier

After a v0.12.0 to v0.13.1 install, both ahoy and ahoy doctor report zero gaps. abcd lint in the same checkout exits 2: rule three-tier-layout errors that the shared working tier (the work folder under the abcd directory) is missing, and decision-durability warns that no committed DECISIONS.md exists there. The install that declares the repository fully set up never creates the tier its own lint requires, so a freshly adopted repository fails lint with nothing on the setup side pointing at it.

To see it: adopt a repository with ahoy install, confirm ahoy reports no gaps, run abcd lint.

Remedy the reporter proposes: Have ahoy detect the missing shared work tier as a safe-autocreate gap and create it on install, or relax the lint rule for repositories ahoy considers healthy.

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610071228213002, a defect against abcd v0.13.1, surface abcd ahoy, abcd ahoy doctor, abcd lint (three-tier-layout, decision-durability).

Evidence:

- rpt-2610071228213002 (the report, kept in the inbox)
- 79e9eba
