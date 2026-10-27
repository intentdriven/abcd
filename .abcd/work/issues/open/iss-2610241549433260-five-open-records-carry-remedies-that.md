---
schema_version: 1
id: "iss-2610241549433260"
slug: "five-open-records-carry-remedies-that"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b05 reading iss-52"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/issues/open"
remedy: "Rewrite each of the five remedies with capture remedy to the successor spelling (abcd library for source, ahoy install for prepare-this-repo), and have the drain hand back a record whose remedy names a retired top-level name, reading the same list as the CLI's stale-usage refusal (retiredTokens in internal/surface/cli/staleusage.go, moved into a core package both read), proven by a drain test on a fixture record whose remedy names abcd source add."
---

Five open records carry remedies that name verbs or pages itd-2610090831227812 retired with no alias on 2026-10-09: iss-25 and iss-52 name abcd source add (iss-52 also source ledger), and iss-92, iss-2608210934566220 and iss-2610040350491915 name prepare-this-repo, so a lane applying any of them targets a command that no longer exists; nothing checks a remedy against the retired names.
