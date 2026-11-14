---
schema_version: 1
id: "iss-2611141812055152"
slug: "the-git-settings-a-drive-holds-across"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/gitenvs.go"
remedy: "judge each such setting: pinned on the command line, held, or shown unread by any command the tree runs, with the table kept beside the held set"
resolution: "A drive holds, beside the settings the run's fingerprint watches, each setting git 2.52 documents as naming a program git starts or as a mode that makes git start one (driveHeldConfig in internal/core/implement/loop/remoteid.go, with the settings left out and why beside it); a read-only merge pins merge.renormalize off; and a log, show or whatchanged whose format the command line does not fix asks the drive's check."
impact: fix
---

The git settings a drive holds across its lanes are not yet judged one by one against git's documentation of the settings that name a program
