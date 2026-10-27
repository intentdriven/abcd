---
schema_version: 1
id: "iss-2610241606415232"
slug: "the-marketplace-catalogue-s-plugin"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "verifying hand-back triage claims, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: ".claude-plugin/marketplace.json"
remedy: "Register the catalogue's plugin description as an identity surface beside plugin.json, so abcd identity reports a drift there and proposes the fix, and align its text with the registered description; prove it with an identity test over a fixture whose catalogue description differs from the manifest's. Grounds: internal, the identity verb's own one-story-everywhere rule."
---

The marketplace catalogue's plugin description (.claude-plugin/marketplace.json: 'Host-agnostic configuration layer for development — a single Go binary, usable as a plugin in compatible agent harnesses.') tells a different story from the plugin manifest's (.claude-plugin/plugin.json: 'For people who know what they want to build and need help shipping it. A single Go binary that carries the why from idea to shipped reality, usable as a plugin in compatible agent harnesses.'), and the catalogue is not among the identity surfaces abcd identity checks, so the drift is invisible to it. Found while verifying the triage of iss-2608220150157505, whose registered surfaces all agree.
