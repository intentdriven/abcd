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
remedy: "A drive states at its start, before the yes, that a reboot, a logout or a crash ends it, and names the one command that resumes it from the drain's state; nothing that starts by itself is installed (ruled 2026-11-25, itd-2611041656024409 decision 1). A test reads the notice and the command at a drive's start."
---

A long unattended run on one machine has nothing that restarts it after the machine reboots, and full-disk encryption can hold the machine at its login screen until someone signs in, so an update or a power cut ends the run and nothing says so. The restart path (itd-2611041549461012) begins with the next session's resume; nothing starts that session.
