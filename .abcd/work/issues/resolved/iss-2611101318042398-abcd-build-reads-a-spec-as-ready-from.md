---
schema_version: 1
id: "iss-2611101318042398"
slug: "abcd-build-reads-a-spec-as-ready-from"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/check.go"
remedy: "Have record-lint refuse a spec that states it is not ready or asks an open question outside its Open Questions section, and have build refuse a step whose spec names a prerequisite intent that has not shipped, naming it; a test builds a spec with a not-READY sentence in its summary and finds it refused."
resolution: "abcd build now refuses a spec whose own text says it is not ready: the readiness gate's spec_body row fails on an open item under the spec's Open Questions, a sentence opening Not READY, or a bold Open: label outside that section, read by spec.Unsettled; record-lint's spec_unsettled rule refuses the last two in an open spec as a blocker. The blocked row reads the open spec's blocked_by beside the intent's, so a prerequisite intent that has not shipped is refused by name. The two open specs on main that said Not READY in prose now ask their questions under Open Questions and stay not READY."
impact: fix
resolved_by:
  commit: "56ef596155417c840d2dec5ee41013cc4a113d44"
---

abcd build reads a spec as READY from its Open Questions section only, so a spec that says in its own prose that it is not READY passes, and a step whose named prerequisite has not shipped is not checked. On 7 Nov at 15:41 and 15:50 build reported itd-2611031342011261 READY, reserved it and made a lane, though spc-2611050023402644's summary says it is not READY until one question is ruled; a session found it before any code was written and released it. On 7 Nov at 16:00 another lane turned out to wait on itd-2610222215485265 shipping, found only by reading. iss-2610312234315875 covers a related gap in intent ready.

## Grounds

- pursued: a spec that says it is not ready, or waits on an unshipped intent, no longer starts a paid run; shown wrong if build next or build starts a lane on such a spec
