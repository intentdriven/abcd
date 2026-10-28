---
schema_version: 1
id: "iss-2610241215547722"
slug: "the-product-thinker-wants-a"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product thinker, 2026-10-24, during an autonomous multi-session test run"
origin: researcher-authored
production_mode: hand-written
found_at: "a self-optimisation capability (no home yet)"
remedy: "Add a self-optimisation loop to abcd: a per-machine, per-account measurement log under ~/.abcd.noindex/ recording what abcd can observe of its own runs (CPU and load per stage, preflight waits and timeouts, sub-agents alive, tokens and spend per agent role and model, review catch rates per reviewer, lane outcomes and fix rounds, reservation waits); a periodic analysis that proposes setup changes from it (load.preflights, the sub-agent ceiling, pace, fix rounds, which models review); each proposal written with its evidence, applied within bounds the person sets and reversible, and the person's own settings kept as ceilings. Drafted by an agent in an autonomous run, from the product thinker's one-line ask; the scope and the bounds are theirs to settle at planning."
---

The product thinker wants a self-optimisation function built into abcd: it monitors CPU utilisation, token use and everything else that can be monitored, and optimises the setup over time. Today every tuning knob is a fixed number a person writes (the preflight slot count, the sub-agent ceiling, the pace window, fix rounds, the review set), and nothing measures the outcomes those knobs drive, so each value is a guess that never improves. The 2026-10-24 test run, seven sessions on one 10-core machine, showed the cost: the load average held near 25, tests timed out, a nitpick lane waited over an hour at the load gate, and the sub-agent cap was negotiated by hand between sessions. iss-2610241214563687 (an adaptive preflight slot count) and iss-2610140652230444 (reviewer catch rates measured per model) are two instances of this one capability.
