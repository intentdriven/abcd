---
schema_version: 1
id: "iss-2611040337182085"
slug: "a-validator-s-brief-says-to-write-the"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "agents/ruthless-reviewer.md"
remedy: "State in the brief how a read-only reviewer returns, or have the driver take the reply as the return; a test renders the brief for an agent with no Write tool."
---

A validator's brief says to write the whole return to a path, but the ruthless-reviewer and security-reviewer agents have no Write tool, so a host-run reviewer has to reply in text or write through its shell.
