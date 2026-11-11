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
resolution: "abcd build now refuses a spec whose own text says it is not ready: the readiness gate's spec_body row fails on an open item under the spec's Open Questions or a sub-heading of it, a sentence opening Not READY, a bold Open label on any line but the one an open question's item opens on, a fence or HTML comment that never closes, or fences or comments whose extent turns on the layout around them, from where every line is read as text, all read by spec.Unsettled, which reads what the rendered page says (frontmatter, inline code spans, fenced blocks and HTML comments are not read; an indented code block is read); record-lint's spec_unsettled rule refuses the last four in an open spec as a blocker. The phrase match is a filter for a mistake; a spec is held by its blocked_by or its open questions. An intent whose own text has a fence or HTML comment that never closes, or fences or comments whose extent turns on the layout around them, is not ready at open_questions. The open-questions reader reads a record's items and their settled markers as the page shows them. A closed spec, and an intent in shipped, superseded or disciplines, is not judged. The blocked row reads the open spec's own blocked_by beside the intent's: a prerequisite is refused only when the spec declares it in its own blocked_by, and a prerequisite named by one step is not read, which stays with iss-2609260932372448. spc-2609202134338445 now declares itd-2610222215485265 in its blocked_by, so a build of itd-2609201916151817 is refused naming that intent. Two intents now read not ready: itd-2611031342011261, because spc-2611050023402644 asks under Open Questions for a product ruling still owed (one cache file, or a per-session rollup the verb only reads); and itd-2610091016262335, already held on main, because spc-2610230609005313 asks three open design questions there."
impact: fix
resolved_by:
  commit: "56ef596155417c840d2dec5ee41013cc4a113d44"
---

abcd build reads a spec as READY from its Open Questions section only, so a spec that says in its own prose that it is not READY passes, and a step whose named prerequisite has not shipped is not checked. On 7 Nov at 15:41 and 15:50 build reported itd-2611031342011261 READY, reserved it and made a lane, though spc-2611050023402644's summary says it is not READY until one question is ruled; a session found it before any code was written and released it. On 7 Nov at 16:00 another lane turned out to wait on itd-2610222215485265 shipping, found only by reading. iss-2610312234315875 covers a related gap in intent ready.

## Grounds

- pursued: a spec that says it is not ready, or waits on an unshipped intent, no longer starts a paid run; shown wrong if build next or build starts a lane on such a spec
