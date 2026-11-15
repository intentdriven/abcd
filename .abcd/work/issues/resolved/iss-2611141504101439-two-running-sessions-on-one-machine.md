---
schema_version: 1
id: "iss-2611141504101439"
slug: "two-running-sessions-on-one-machine"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "two sessions seen with one name"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "Make the hold self-healing instead of a single attempt: on each prompt the hook re-asserts this session's name under the register's lock (find its entry by session id; when it is missing, re-make it for the process now running the session; when another live session holds the name, the later claim takes the next free name and its first output says which name it now has), treat a name as free only when its last holder's process is proven gone by pid and start time, and on a timeout keep the title and allocate nothing. Grounds: RFC 6762 section 9 (a name in use is defended: on a conflict the later claimant renames and the person is told; the same data from the same holder is no conflict), ZooKeeper's rule that a client re-creates its ephemeral state after its session is lost, and flock(2) for liveness that needs no record. Proven by register tests: a resume whose hold fails is held at its next prompt; two live sessions with one name end with two names and one notice; a hook that times out changes nothing."
resolution: "Every prompt now re-asserts the session's hold on its name: a missing entry is made again for the process now running the session, and when another running session holds the name the later claim takes the next free name and that prompt says so once. A lock or process-table read that runs out of time changes nothing."
impact: fix
resolved_by:
  commit: "aff97817c"
---

Two running sessions on one machine carried the same persona name for four days. One session was resumed after every session on the machine had ended at once; it kept its title, and the register held no live entry for it afterwards. A new session started two minutes later, found the name free and was given it. The register is written at session start and nowhere re-checks a held name: a compaction of a titled session does nothing, and no prompt re-asserts the hold, so a hold that is not made at the one resume call is never made, and nothing notices two live sessions with one name. Why the hold was not made at that resume is not established (the hook's note from that start is gone); the lock's two-second wait and the one-second bound on the process-table read, with many sessions starting together, are the candidates. A message addressed to the name then reaches either session.
