---
schema_version: 1
id: "iss-2610091937218987"
slug: "run-the-live-network-cut-lab-for-the-lost-connection-fix"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "building the fix for iss-2610080620372731 on 2026-10-09"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/outage.go"
remedy: "After the next patch release is tagged, mint a lab with abcd lab mint, run one autonomous run on a scratch repository, cut the network (and separately block the model service) for a few minutes at a time with the product thinker's consent, record what each kind of loss did against the settled schedule, and capture any divergence as its own issue."
refines: [iss-2610080620372731]
---

The lost-connection fix of iss-2610080620372731 is proven by unit tests with an injected clock and prober, not yet by a live run: the record asks for a lab that cuts the network during a real autonomous run and shows each kind of loss (the host's own turn, a sub-agent, a tool's network call, and the model service) waited out on the one shared probe, the other lanes held only at network steps, a dead or yielded lane restarted from its last commit with its edits kept for review, and the 8-hour give-up ending in one notification. The product thinker ruled on 2026-10-09 that this lab runs after the next patch release is cut, since it switches off the machine's network for a few minutes.
