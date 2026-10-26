---
schema_version: 1
id: "iss-2610252214382588"
slug: "35-drain-md-252-253-says-a-pull-request"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S06"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/35-drain.md"
remedy: "Rewrite step 5 so a conflict on declared generated files alone, or one the checkout cannot reproduce, re-enters the sync on the drain's next move and any other is refused by the run's step, or replace step 5 with a pointer to commands/drain.md:399-415."
---

35-drain.md:252-253 says a pull request that turns conflicting after arming is refused by the run's own step 'since the drain's part ends at the open pull request', but the drain's move re-enters the sync for a parked lane whose conflict is confined to declared generated files or not reproducible here (internal/core/implement/loop/drain_finish.go:274-283), as commands/drain.md:407-412 and the chapter's own :413-416 say; :413-415 also omits the not-reproducible case commands/implement.md:471-472 names.
