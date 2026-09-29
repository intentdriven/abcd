---
schema_version: 1
id: "iss-2608271804188295"
slug: "itd-70-is-a-machine-emitted-stub-not-a-press-release"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/development/intents/drafts/itd-70-launch-release-retention-newest-per-line.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling F owed to the product thinker: drafts/itd-70 is still the eleven-line stub whose provenance line cites spc-75, which exists nowhere. iss-194 has since resolved (the brief states the prune is computed and previewed, not executed), so deleting the draft now leaves iss-282 alone carrying retention. Flesh it into a press release or delete it. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

drafts/itd-70 is a machine-emitted stub, not a press-release intent: no press release, no scope, no acceptance criteria, and a provenance line citing spc-75 which exists nowhere (iss-239's class). The drafts lint sanctions its frontmatter shape, so the only defensible action is editorial — flesh it into a press release before anyone plans it, or delete it and let open iss-194 carry the release-retention concern.