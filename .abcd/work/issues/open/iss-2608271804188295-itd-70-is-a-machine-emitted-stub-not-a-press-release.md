---
schema_version: 1
id: "iss-2608271804188295"
slug: "itd-70-is-a-machine-emitted-stub-not-a-press-release"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/development/intents/drafts/itd-70-launch-release-retention-newest-per-line.md"
remedy: "Waits on the itd-70 planning interview (ruling J21 of 2026-09-29: automate retention, plan itd-70, which answers flesh-or-delete with flesh): replace the stub in `.abcd/development/intents/drafts/itd-70-launch-release-retention-newest-per-line.md` with a press release drawn from iss-282 (newest per MAJOR.MINOR line kept, the prune executed rather than only previewed), drop the provenance line citing the non-existent spc-75, and resolve this record in that change, proven by record-lint green."
deferred_after: "v0.11.1"
deferral_reason: "ruling F owed to the product thinker: drafts/itd-70 is still the eleven-line stub whose provenance line cites spc-75, which exists nowhere. iss-194 has since resolved (the brief states the prune is computed and previewed, not executed), so deleting the draft now leaves iss-282 alone carrying retention. Flesh it into a press release or delete it. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

drafts/itd-70 is a machine-emitted stub, not a press-release intent: no press release, no scope, no acceptance criteria, and a provenance line citing spc-75 which exists nowhere (iss-239's class). The drafts lint sanctions its frontmatter shape, so the only defensible action is editorial — flesh it into a press release before anyone plans it, or delete it and let open iss-194 carry the release-retention concern.

## Remedy grounds (2026-09-29)

- J21 rules to plan itd-70, so deleting the draft is off the table and the remaining work is the fleshing the planning interview performs; iss-194 has resolved, leaving iss-282 as the open carrier the press release draws from.
- No outside-practice check: an editorial repair of an internal record.
- Rejected: deleting the draft, which the J21 ruling overtakes.
