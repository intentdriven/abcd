---
schema_version: 1
id: "iss-2611061852271307"
slug: "drive-preconditions-unseen-by-the-dry"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "first real drive of a drain ahead of a long run"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain_drive.go"
remedy: "Have the dry run run the drive's preconditions and report each as a line, in the order the drive refuses them; have the refusal name the file and print the exact change; and let a pin stand for host sessions without blocking a drive, or say why it must go."
---

abcd drain --dry-run showed nothing wrong, then the first abcd drain --drive on the same machine refused to start: every role it needs was pinned to the host, which has no session to run in. The dry run reads none of the conditions the drive refuses on, so a person preparing a long run learns about them only at the moment of starting it. The refusal also says to remove the pin without saying which file holds it or what to change, and the pin belongs to the machine's configuration, where it is correct for a build run inside a host session.
