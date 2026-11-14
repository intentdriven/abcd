---
schema_version: 1
id: "iss-2611140902531015"
slug: "a-run-taken-over-from-a-terminal-with"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "drive restart after a stop"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/control.go"
remedy: "In the driver claim (internal/core/implement/loop/control.go, the contend at the alive-driver branch), treat a recorded driver that is an ancestor process of the claiming drive, in the same terminal session, as this session's own, so the drive takes the run; and have resume --take say that a drive must be started from the same terminal, or record a driver that ends with the command. Proven by a loop test in which a run's recorded driver is the test process's parent and a drive claims it."
---

A run taken over from a terminal with implement resume --take names that terminal's login process as its driver. A drive started afterwards from the same terminal reads the run as driven by another live session and waits on it without end, opening no other lane, because the login process lives as long as the drive's own terminal does. Seen on 14 Nov: the take-over was the remedy the drive's own stop message named, and the drive then logged one wait on the run and nothing else.
