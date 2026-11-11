---
schema_version: 1
id: "iss-2611111850449220"
slug: "scan-cleanup-cut-short-by-a-second"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "merge queue runs of 11 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/preflight-scans.sh"
remedy: "Each signal trap ignores HUP, INT and TERM before it exits, so the removals run to their end; a test sends a second hangup from inside the first removal and requires nothing left behind."
---

The preflight scan script's cleanup can be cut short by a second signal. Its HUP, INT and TERM traps exit with the signal's code and the EXIT trap then removes the fork point's copy, the gitleaks directory and the directory govulncheck was built in, one after another. A second signal that arrives while those removals run takes the same trap, which exits at once: the exit code is still the first trap's, and whatever was not yet removed stays in TMPDIR. Seen in the merge queue's Linux test job three times on 11 Nov 2026: TestVulncheckInterruptedLeavesNoCopyBehind/HUP reported the build directory left behind twice and the fork point's copy with it once, with the exit code as expected each time, and two pull requests were removed from the queue. Not reproduced on macOS by repeating the test; where the second hangup comes from on Linux is not established (a shell in a command substitution passing a hangup on to its process group is the likely source, read from how bash behaves, not observed).
