---
schema_version: 1
id: "iss-2610241214563687"
slug: "the-product-thinker-wants-the-number-of"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "drain session abcd-0d, 2026-10-24, setting load.preflights on a busy machine"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_slots.go"
remedy: "Make the preflight slot count adaptive: the load gate keeps a per-machine log of each preflight stage's wait, run time, load average and timeouts (under ~/.abcd.noindex/), and adjusts load.preflights within 1..MaxPreflightSlots from that evidence, up while stages finish well inside their timeouts with the load below the core count and down when tests time out or the load stays far above it, writing each change and its grounds where the person can read and override it; the person's own setting stays the ceiling."
---

The product thinker wants the number of preflights a machine runs at once to follow the evidence, not stay fixed. On 2026-10-24 they set load.preflights to 2 in ~/.abcd.noindex/config.json, which applies to every abcd-managed repository on the machine, and asked that load monitoring keep it right: raised when the machine has room, lowered when it is overloaded. Today the count is a fixed number a person writes (default 1, at most 2), and the gate measures load only to decide whether to wait, never to learn: nothing records how long stages waited or ran, whether tests timed out, or what the load was, so no one can tell whether 2 is too many or too few.

Evidence from the same run, 2026-10-25/26, on a 10-core machine with 16 GB: with load.preflights at 2 and two preflights in their test stages at once, the one-minute load reached 42 and package tests timed out at their per-package limits, so the setting went back to 1 the same evening. With 1, and with the machine's endpoint-security scanner switched off (it had been using about four cores of its own), this session's two whole preflights (the repairs of PRs 1110 and 1116) each passed at the first attempt. A fixed number is therefore wrong in both directions: 2 overloaded the machine while the scanner ran, and the right number changed when the scanner stopped, which no person-written setting tracks.
