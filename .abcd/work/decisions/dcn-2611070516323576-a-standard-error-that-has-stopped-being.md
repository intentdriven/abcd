---
id: dcn-2611070516323576
slug: a-standard-error-that-has-stopped-being
date: 2026-11-07
---
2026-11-07 — A standard error that has stopped being read can keep a drive's second signal from ending the agent's process group, and this is carried past this cut to the follow-up bugfix release. A signal's handling writes its line before it reads the next signal or ends the context, so a write that blocks holds up the kill, and only a forced kill then ends the drive, leaving the agent's group running. It predates self-update, it takes a terminal or pipe that has stopped reading at the moment a person asks for a stop, and a correct fix changes the order of the signal path, which needs its own review. The follow-up: end the context and arm the exit before saying the second line, and say the first line where it cannot hold up reading the second signal.
