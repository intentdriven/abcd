---
schema_version: 1
id: "iss-2611121517269047"
slug: "a-drive-keeps-a-lane-it-passed-over"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal.go"
remedy: "Before a drive ends on passed lanes, judge each once more and take back one it may now save"
---

A drive keeps a lane it passed over, since it may not save the run, passed for the whole drive whatever the judge's reason, so a lane passed on a reason that can change while the drive runs, such as a process table that could not be read or a session that could not be found, stays passed when the drive ends on passed lanes, and the drive stops short at exit 3 over a lane it may by then save
