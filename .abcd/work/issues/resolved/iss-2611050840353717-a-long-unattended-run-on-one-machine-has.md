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
remedy: "A drive's full plan, on ? and in its log, says that a reboot, a logout or a crash ends the drive and names the one command that resumes it from the drain's state, and the drive prints that command once after the yes; the start keeps its seven lines, and nothing that starts by itself is installed (ruled 2026-11-25, itd-2611041656024409 decision 1; built by step 3 of spc-2611251810508418). A test reads the command on ? and after the yes at a drive's start."
related_intents: [itd-2611251809580662]
resolution: "a drive's start now fixes its end before the yes, seven days on or the repository's drive.until_days; its plan names that a reboot, a logout or a crash ends the drive and the one command that carries it on to the same end, prints that command once after the yes, and heads the drive's log; re-execution passes the end as an explicit --until"
impact: additive
resolved_by:
  intent: "itd-2611251809580662"
  spec: "spc-2611251810508418"
---

A long unattended run on one machine has nothing that restarts it after the machine reboots, and full-disk encryption can hold the machine at its login screen until someone signs in, so an update or a power cut ends the run and nothing says so. The restart path (itd-2611041549461012) begins with the next session's resume; nothing starts that session.
