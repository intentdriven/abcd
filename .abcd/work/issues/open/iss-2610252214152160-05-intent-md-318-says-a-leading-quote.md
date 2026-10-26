---
schema_version: 1
id: "iss-2610252214152160"
slug: "05-intent-md-318-says-a-leading-quote"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S02"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Replace the clause with the lone-word refusal, or point the row at commands/intent.md § Create a draft."
---

05-intent.md:318 says 'A leading quote always creates — never falls through to bare render'; a quoted lone word is refused at exit 2 (internal/surface/cli/cli.go:2772, commands/intent.md:119). Found by the 2026-08-23 crosscheck and never fixed.
