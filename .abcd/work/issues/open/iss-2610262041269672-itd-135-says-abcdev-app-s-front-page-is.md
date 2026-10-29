---
schema_version: 1
id: "iss-2610262041269672"
slug: "itd-135-says-abcdev-app-s-front-page-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1000 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-135-iris-opens-abcdev-app-and-understands.md:22"
remedy: "In 10-site.md, remove the opening design-target mark (lines 3-11) or rewrite it to say abcdev.app serves the rendered site at its root and the MkDocs rendering at /docs/, since the release chain deploys on every full release."
related_intents: [itd-135]
finding_ref: cfk-44f9b97a8d0b589e
---

itd-135 says abcdev.app's front page is now the rendered landing story; the site chapter says abcdev.app still serves the MkDocs rendering at its root

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-135-iris-opens-abcdev-app-and-understands.md:22` — "Now the front page is that story"
- End B: `.abcd/development/brief/05-internals/10-site.md:6` — "abcdev.app still serves the MkDocs rendering of `docs/` at its root"

itd-135 is shipped, and its press release says that where abcdev.app used to greet a visitor with a documentation filing system, the front page is now the landing story. The site chapter marks that passage as a design target: abcdev.app still serves the MkDocs rendering of `docs/` at its root, and only the first production deploy from a tag moves it. The live root cannot be both the landing page and the MkDocs rendering.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1000.
