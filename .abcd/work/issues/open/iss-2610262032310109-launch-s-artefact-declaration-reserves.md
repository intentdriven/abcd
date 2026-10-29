---
schema_version: 1
id: "iss-2610262032310109"
slug: "launch-s-artefact-declaration-reserves"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 280 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:35"
remedy: "In 04-launch.md line 35 replace 'reserves the release-rendered site's opt-in, read and not yet acted on' with the present state: the key is read and validated and nothing consumes it, because the release-rendered site is set up by the site verb (22-site.md); decide separately whether to retire the key (internal/core/launch/artefact.go)."
finding_ref: cfk-574e18fe83dc9685
---

launch's artefact declaration reserves an inert opt-in for the release-rendered site; site describes a shipped setup that already writes a workflow rendering the site from each published release

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:35` — "reserves the release-rendered site's opt-in, read and not yet acted on"
- End B: `.abcd/development/brief/04-surfaces/22-site.md:55` — "a workflow that renders the site from each published release with abcd's checksum- and attestation-verified binary"

04-launch.md says the artefact declaration's site key reserves the release-rendered site's opt-in, read and not yet acted on. 22-site.md describes a shipped setup verb that writes the composition declaration and a workflow rendering the site from each published release. The release-rendered site is therefore already acted on, and it is opted into through the site verb, not artefact.json. Two declarations claim the same opt-in, and the one launch names does nothing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 280.
