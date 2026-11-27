---
id: dcn-2611260641427714
slug: defer-the-flake-in-cmd-precheck-audits
date: 2026-11-26
---
2026-11-26 — Defer the flake in cmd/precheck-audits TestAnInterruptStopsEverything out of the night branch's landing. Grounds: it failed once in make precheck on 6bda675a5 (main_test.go:553, strconv.Atoi parsing "") and passed three runs alone; the branch changes nothing under cmd/precheck-audits. The cause is in the test: its watcher stops once both pid files exist, before the shell has written the pid, so gone() reads an empty file. The fix (wait for a non-empty pid, as waitForFile could) belongs in a change of its own on main. Reverse by fixing it in this branch instead.
