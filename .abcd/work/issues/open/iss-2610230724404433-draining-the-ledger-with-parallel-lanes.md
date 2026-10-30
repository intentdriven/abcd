---
schema_version: 1
id: "iss-2610230724404433"
slug: "draining-the-ledger-with-parallel-lanes"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "parallel ledger drain, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/drain.md"
remedy: "Give abcd drain a --lanes <n> mode that opens up to n issue lanes at once, picking the next eligible issues whose found_at overlaps no live reservation and counting their agents against one ceiling; add implement step --all (or a compact status line per run) so a host drives every run in progress with one call; and let the cap refusal offer the setting as a verb that writes and validates ~/.abcd.noindex/config.json rather than naming a file to hand-edit."
---

Draining the ledger with parallel lanes on 2026-10-23 needed five things done by hand, because no abcd verb does them: (1) abcd drain opens one lane at a time, so the extra lanes were started as separate abcd build <iss-N> runs, and the issues for them were picked by hand, checking each found_at against every live reservation (two picks were then refused for overlapping a peer's footprint); (2) driving five runs meant a hand-written helper, .abcd/.work.local/scratch/drain-step.sh, that steps one run until it awaits an agent and prints a one-line summary, since implement step prints a long JSON document per call and has no form that steps every run in progress; (3) the per-person reservation cap (10) was already passed by one person's sessions across two accounts, and raising it meant hand-writing ~/.abcd.noindex/config.json with reservation.max_live, a file nothing creates or validates in advance; (4) each implementer had to be told to run make preflight through implement load --wait --timeout 8h, as iss-2610222251373104 records; (5) the host's wake-up prompt carried the list of run ids by hand, since nothing lists the runs a drain session is driving.
