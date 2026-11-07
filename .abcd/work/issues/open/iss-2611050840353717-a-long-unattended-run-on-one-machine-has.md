---
schema_version: 1
id: "iss-2611050840353717"
slug: "a-long-unattended-run-on-one-machine-has"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "coordinating build lanes, 2026-11-05"
origin: researcher-authored
production_mode: hand-written
found_at: "a long unattended drive on one machine"
remedy: "A drive states this at its start and says how to resume. Where the platform allows it, offer a way to resume on login, off until a person sets it up. A test reads the notice at a drive's start."
---

A long unattended run on one machine has nothing that restarts it after the machine reboots, and full-disk encryption can hold the machine at its login screen until someone signs in, so an update or a power cut ends the run and nothing says so. The restart path (itd-2611041549461012) begins with the next session's resume; nothing starts that session.
