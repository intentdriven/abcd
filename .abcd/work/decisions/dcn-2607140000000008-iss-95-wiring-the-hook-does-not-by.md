---
id: dcn-2607140000000008
slug: iss-95-wiring-the-hook-does-not-by
date: 2026-07-14
---
2026-07-14 (M1) — iss-95: wiring the hook does NOT by itself start the clock. `history.
Capture` requires `~/.abcd/history/<root-sha>/transcripts/` to already exist and
deliberately never creates it (the `ownedDirsReal` symlink-safety discipline); `ahoy
install` bootstraps it. On a machine where install has not run — INCLUDING THIS ONE,
where `~/.abcd/` does not exist — `hook session-end` fails closed, logs to stderr, exits
0, and captures nothing, silently. That is exactly itd-89's failure mode (a hook that
looks wired while the corpus never accrues). Decision owed: hook self-bootstraps (changes
Capture's precondition and has the hook create dirs, which the symlink discipline avoids)
vs. `ahoy install` stays the sanctioned bootstrap and the not-installed case is made LOUD
(ahoy doctor already flags `history.bootstrap_missing`). iss-96 records the adjacent point:
automatic capture makes the scanner's secret-pattern coverage load-bearing — it catches
anchored tokens (AKIA…, ghp_, sk-ant-) and home paths but not unanchored high-entropy
values (a bare 40-char AWS secret, a prefixless token), so consider entropy detection or
the gitleaks adapter for the transcript path.
