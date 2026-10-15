---
id: dcn-2608180000000004
slug: two-adversarial-reviews-of-the-adr-42
date: 2026-08-18
---
2026-08-18 — Two adversarial reviews of the adr-42 implementation; three findings that
changed the code, all reproduced against the binary first. (1) DoS, 23.6s at the 1 MiB cap:
maxSpeculativeWindow bounds the token COUNT and an execute-a-string payload is ONE token of
unbounded length, so 64 starts each re-tokenized a 1 MiB payload. Two more bounds added — a
payload-BYTES budget and a whole-line START budget, both per Check, after a per-segment byte
budget still left 1,200 short segments at 8.4s. The lesson generalises: every per-item bound
needs a whole-input bound behind it, and the ADR had said in advance that the benchmark must
exercise the payload path — the first bound test pinned the cheaper half anyway. The bound
test is now four adversarial shapes, each verified by mutation to fail on the bound it names.
(2) FALSE BLOCK, and a suppression behind it: scanExecString ran to the end of the segment, so
`flock /tmp/lock /bin/echo -c "<hazard>"` read a `-c` that real flock passes to /bin/echo as an
argument (verified: it prints them). That both blocked a harmless command AND made the segment
look understood, switching Tier 2 off for it — a `flock` prefix was a one-token off switch,
the payload-level twin of the per-line suppression decision 4 rejects. Fixed with a per-verb
count of the operands the verb owns before the launched command begins; only flock has one,
because su genuinely permutes (`su root /bin/echo -c 'echo SEVEN'` prints SEVEN). This
falsified the code's own claim that a mis-parse "never yields a false block".
(3) The wrapper probe would have FAILED ON CI: `chrt -f 1` needs CAP_SYS_NICE and a GitHub
runner is unprivileged, and the per-wrapper needsRoot blanket also hid the nsenter -W
counterexample the file exists for. Replaced with a DERIVED control probe (if the wrapper
cannot run its own baseline here, nothing it says is evidence) plus per-flag reasons. The
candidate list — itself a hand-maintained enumeration, i.e. the CWE-184 shape inside the test
written to defeat it — now comes from the binary's own --help, which is trusted to ENUMERATE
and never to CLASSIFY. That immediately found three real misses: nice --adjustment, runuser -w
and runuser -s, each degrading a precise block to a warn. 44 classifications became 121.
Also: every synthetic id that contributed is now listed in Matches (a Tier 2 fire used to lose
the slot to any earlier payload warn and vanish — which the warn-rate gate counts, so the blind
spot hid itself); the cap warn no longer fires when only losslessly-skipped tokens remain; the
warn ceiling is an absolute 2 rather than a 1% rate that permitted 9; matchesAny deleted as
dead; and the three surfaces no longer list "launched through a wrapper outside the known set"
as something an allow does not see, which is the case this change converts to a warn.
