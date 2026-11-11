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
---

abcd build reads a spec as READY from its Open Questions section only, so a spec that says in its own prose that it is not READY passes, and a step whose named prerequisite has not shipped is not checked. On 7 Nov at 15:41 and 15:50 build reported itd-2611031342011261 READY, reserved it and made a lane, though spc-2611050023402644's summary says it is not READY until one question is ruled; a session found it before any code was written and released it. On 7 Nov at 16:00 another lane turned out to wait on itd-2610222215485265 shipping, found only by reading. iss-2610312234315875 covers a related gap in intent ready.
