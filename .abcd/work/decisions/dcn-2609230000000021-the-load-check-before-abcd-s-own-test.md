---
id: dcn-2609230000000021
slug: the-load-check-before-abcd-s-own-test
date: 2026-09-23
---
2026-09-23 — The load check before abcd's own test lanes warns and never refuses (product thinker, planning interview for itd-2609231434459890; spc-2609231542463113, Decision 2). This changes the fourth condition of the run A interview ruling on iss-2609210828122412, which had the check refusing. The check names foreign sustained CPU (a near-full core for longer than 30 minutes) and an extreme load (four times the online cores), both limits overridable in the caller's `~/.abcd/load-limits`; it names the caller's own strays with a remedy and counts other accounts' processes without naming them; it writes a run-log event; it runs once at the start of `make preflight` and of the eval harness; CI skips it with a stated reason; it always exits 0, and its findings are minor.
