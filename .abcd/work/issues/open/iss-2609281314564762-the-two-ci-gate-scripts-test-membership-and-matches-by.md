---
schema_version: 1
id: "iss-2609281314564762"
slug: "the-two-ci-gate-scripts-test-membership-and-matches-by"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25 (lane drainScr, full-history before/after runs)"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-issue-resolution.sh"
---

The two CI gate scripts test membership and matches by piping printf into grep -q under set -o pipefail, and that pipeline is a race whenever printf needs more than one write: grep -q exits at its first match, printf's next write takes SIGPIPE, the pipeline returns 141, and the test reads as no match. Under load the race loses at a few KiB; past one pipe buffer (64 KiB) it loses every time. Measured: a 50 KiB id list missed 0 of 30 runs and a 70 KiB one 30 of 30; a here-string missed none. In scripts/check-issue-resolution.sh the RS001 membership test against the ids entering a terminal folder, RS004's declared-id test, and RS005's shipped test all take this shape, so two full-history runs of the gate over the same range at the same commits differed by 47 violations (39 RS001 refusals of records that did enter resolved/, seven RS004 refusals of ids a 190-line Refs: block declared, one RS005), and the base-listing membership test the lane adds for iss-2609012047551175 would read a record terminal at the base as absent on a real ledger, failing open. In scripts/check-attribution.sh check_text pipes a whole commit message or pull-request body into grep -q for the tool-footer, Co-authored-by and trailer checks, so a body longer than the buffer whose banned line is matched before the last write can pass. Fix: feed grep from a here-string, which bash writes in full before grep reads, so no writer is left to take SIGPIPE.
