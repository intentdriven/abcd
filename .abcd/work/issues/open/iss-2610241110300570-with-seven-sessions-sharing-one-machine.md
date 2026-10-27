---
schema_version: 1
id: "iss-2610241110300570"
slug: "with-seven-sessions-sharing-one-machine"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "drain session abcd-0d, 2026-10-24, the first drain lane"
origin: researcher-authored
production_mode: hand-written
found_at: "the preflight load gate (implement load)"
remedy: "Record per lane the minutes its definition of done spent waiting at the load gate, report it in the drain's counts and the run's state, and let the drain plan its lanes against that measured wait instead of the fixed 120-minute window."
---

With seven sessions sharing one machine, a drain lane's definition of done waited over an hour at the preflight load gate ("waiting for capacity: 3 make preflights are running"), and the drain's 120-minute window elapsed while its first lane was still in progress. The gate admits one preflight at a time by default, so each session's preflight queues behind every other's: on a busy account a one-line nitpick fix takes hours to reach its reviewers. Nothing measures or reports that wait, so a run cannot tell a lane stuck at the gate from one still working.
