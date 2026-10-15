---
id: dcn-2608180000000003
slug: adr-42-parts-b-and-c-built-iss-272
date: 2026-08-18
---
2026-08-18 — adr-42 parts B and C built; iss-272 closed. Part B named fourteen wrappers
(nice setsid stdbuf ionice eatmydata proxychains chrt taskset unshare nsenter flock chroot
runuser busybox), which upgrades each from part A's loud warn to a precise Tier 1 block.
Every flag list is DERIVED BY PROBING the installed binary in a test, never from --help,
and the probe caught two errors before they shipped: `nsenter -W/--wd` consumes NOTHING
while `unshare -w/--wd` — same letter, same package, same version — is required-argument
(listing nsenter's would have made the walk step over the COMMAND, a miss the table would
have INVENTED); and `taskset -c` is a format switch for the MASK operand, not a value flag.
A flag this environment cannot classify is listed as unprobed WITH ITS REASON (setsid --ctty
needs a controlling terminal, chrt --sched-runtime needs SCHED_DEADLINE, nsenter/chroot/
runuser need privileges) — silence is what gh-299 shipped. Part C added an exec-string TABLE
for su/runuser/script/flock, not a shellCPayload generalisation, which would have resolved
the short spellings free and silently allowed six long ones (su --command, --command=,
--session-command, runuser --command, script --command, flock --command); each has a test.
Classification runs on the raw token chain before commandOf because runuser and flock are
also wrappers. Recorded limit kept honest: `su -c` runs the TARGET user's login shell,
overridable with -s, so the payload is not guaranteed POSIX grammar — a false-negative cost
only. All ten of iss-272's bypasses now produce a verdict; the acceptance test asserts that
in the issue's own terms.
