---
schema_version: 1
id: "iss-2609061525437969"
slug: "the-ai-assistance-disclosure-convention-covers-commits-only"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "release-gate adoption in a managed public repo, 2026-09-06"
origin: researcher-authored
production_mode: hand-written
found_at: "rules (COMMITTING domain), hooks, internal (guard)"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Extend Assisted-by disclosure to PR bodies and issue text, with gh-CLI guard coverage?"
---

The AI-assistance disclosure convention covers commits only: the COMMITTING rule requires an Assisted-by trailer on AI-assisted commits, but says nothing about pull-request descriptions or issue text, and nothing checks either. In a managed repo two PRs went out with the trailer on every commit and none on the PR body, which the maintainer read as an oversight. Extend the convention to PR descriptions (and issue text written by an agent), teach the guard's GitHub-CLI coverage to require the trailer there, and make the same guard refuse session links and other tool footers — the two halves of one disclosure rule: say a tool assisted, never link the session.
