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
resolution: "Every membership and match test in scripts/check-issue-resolution.sh and scripts/check-attribution.sh reads a here-string instead of a printf pipe, and RS006's frontmatter reader reads to the end instead of exiting early; the cases suites' own helpers take the same change. Proved by two cases, each failing against the previous scripts: a message declaring 1,100 long ids (about 70 KiB) is read whole by RS004 (the previous gate refused 39 of its declared ids), and a Co-authored-by line at the top of a 200 KiB pull-request body is refused (the previous gate accepted it). The RS006 reader half landed in eff2eefda, proved by a resolved record of about 200 KiB passing (the previous gate ended at exit 141)."
impact: internal
resolved_by:
  commit: "5b7c79426"
---

The two CI gate scripts test membership and matches by piping printf into grep -q under set -o pipefail, and that pipeline is a race whenever printf needs more than one write: grep -q exits at its first match, printf's next write takes SIGPIPE, the pipeline returns 141, and the test reads as no match. Under load the race loses at a few KiB; past one pipe buffer (64 KiB) it loses every time. Measured: a 50 KiB id list missed 0 of 30 runs and a 70 KiB one 30 of 30; a here-string missed none. In scripts/check-issue-resolution.sh the RS001 membership test against the ids entering a terminal folder, RS004's declared-id test, and RS005's shipped test all take this shape, so two full-history runs of the gate over the same range at the same commits differed by 47 violations (39 RS001 refusals of records that did enter resolved/, seven RS004 refusals of ids a 190-line Refs: block declared, one RS005), and the base-listing membership test the lane adds for iss-2609012047551175 would read a record terminal at the base as absent on a real ledger, failing open. In scripts/check-attribution.sh check_text pipes a whole commit message or pull-request body into grep -q for the tool-footer, Co-authored-by and trailer checks, so a body longer than the buffer whose banned line is matched before the last write can pass. Fix: feed grep from a here-string, which bash writes in full before grep reads, so no writer is left to take SIGPIPE.

## Grounds

- pursued: the gates give the same verdict on the same range whatever the machine load or input size; two full-history runs at the same commits disagreeing again would show it wrong
