---
schema_version: 1
id: "iss-2609240227595724"
slug: "itd-33-runs-against-three-later-rulings"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A, planning briefs"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-33-agent-communication-infrastructure.md"
---

itd-33 (draft, written 2026-07-06) runs against three later rulings, which its planning interview has to settle. (1) adr-2609091248200336 (2026-09-09): agent and session state is machine-scoped under ~/.abcd/ keyed on the root commit, and the checkout holds only its declared tiers; itd-33 puts per-machine live state and locks (active-work.json, *.lock) in a new .abcd/coordination/ directory inside the checkout, which is no declared tier, beside a committed audit/ log. The shipped claim of itd-2609221656373558 already keeps its run store outside the repository. (2) The basics-plus-adapter principle (principles/basics-built-in-adapters-bring-power.md, 2026-09-15): every capability ships a basic form and offers an opt-in adapter for the full form; itd-33 is always on with no opt-out and rules out any external route, so it offers no adapter. This one is a tension in spirit: the principle's letter does not forbid an always-on basic. (3) itd-2609221656361680 Decision 4, identity in the inbox and a fingerprint in the record: anything committed carries the root-commit key and a generic description; itd-33's committed audit log names the person from git config on every agent_registered event and in resolved_by_human. Decision 4's letter governs the managed-repository reports; applying it here is by extension. The register ruling (itd-2609150819440345 Decision 1, the register is where claims live) overlaps its claim model too. Distinct from iss-2608230943533581, which asks for the coordination SOTA sweep.
