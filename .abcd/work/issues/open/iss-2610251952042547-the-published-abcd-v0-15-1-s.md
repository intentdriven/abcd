---
schema_version: 1
id: "iss-2610251952042547"
slug: "the-published-abcd-v0-15-1-s"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "coordinating sessions during quiet hours, 2026-10-25"
origin: researcher-authored
production_mode: hand-written
found_at: "the UserPromptSubmit prompt-router hook"
remedy: "Bound the prompt-router hook by time as its one-second contract says: measure where the time goes on a loaded machine (rule layer reads, recall matching, git calls), stop at the deadline with whatever it has, and say on stderr that it stopped short; report hook runs that overrun in abcd status so a session sees the cost it adds to every prompt."
---

The published abcd v0.15.1's prompt-router hook ran for 7 to 10 seconds at about 99% CPU per prompt in a busy session on 2026-10-25 (one sample: pid 98266, 9 s; another: pid 370, still running at 7 s), on a 10-core machine at load 25 where an endpoint scanner already takes about four cores. The hook has a one-second contract (TestThePromptHookReturnsInsideItsSecond), but on a loaded machine nothing holds it to that: a session that receives many messages pays a full core for most of every ten seconds, which slows every preflight on the machine and delays each reply.
