---
schema_version: 1
id: "iss-2609061525437969"
slug: "the-ai-assistance-disclosure-convention"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "release-gate adoption in a managed public repo, 2026-09-06"
origin: researcher-authored
production_mode: hand-written
found_at: "rules (COMMITTING domain), hooks, internal (guard)"
remedy: "Adopted by the product thinker on 2026-10-26 (dcn-2610261152329451), narrowed to pull-request bodies: the shell guard's forge-text check refuses a gh pr create or gh pr edit whose body or body file carries no Assisted-by line in one of the three accepted forms, judged with internal/core/assistedby's grammar after fenced blocks are stripped as the CI body arm strips them, and the bundled COMMITTING rule names pull-request bodies beside commits. Agent-written issue text is not part of the ruling. Prove with guard tests that refuse a create whose body lacks the line and pass one carrying each accepted form."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lapsed-deferral triage, run A 2026-09-29): Should the bundled COMMITTING rule require the Assisted-by line on pull-request bodies and agent-written issue text in managed repos, with the shell guard checking the forge CLI's create commands for it? The session-link half is already in force (the bundled outward-text rule and the outbound policy), and this repository enforces the body trailer in CI only (scripts/check-attribution.sh, body arm)."
---

The AI-assistance disclosure convention covers commits only: the COMMITTING rule requires an Assisted-by trailer on AI-assisted commits, but says nothing about pull-request descriptions or issue text, and nothing checks either. In a managed repo two PRs went out with the trailer on every commit and none on the PR body, which the maintainer read as an oversight. Extend the convention to PR descriptions (and issue text written by an agent), teach the guard's GitHub-CLI coverage to require the trailer there, and make the same guard refuse session links and other tool footers — the two halves of one disclosure rule: say a tool assisted, never link the session.

## Remedy grounds (2026-09-29)

- The session-link half is in force (the outward-text rule and the outbound policy); only the disclosure line on bodies and issues is missing, and the guard is where creation is seen before it is public.
- SOTA check: Ghostty's AI policy (https://github.com/ghostty-org/ghostty/blob/main/AI_POLICY.md, read 2026-09-29) requires disclosure of any AI use in pull requests and issues, while the Linux kernel's coding-assistants page (https://docs.kernel.org/process/coding-assistants.html, read 2026-09-29) specifies the commit tag only, so disclosure on bodies is established practice but not universal.
- Rejected: a CI-only check, which judges a body after the forge has kept it.

## Recurrence (2026-10-26)

- In abcd's own repository a pull request went out with the trailer on every commit and none on its body. The shell guard's forge-text check read the body before it posted and passed it, because it judges leaks only. CI's attribution job then failed on the body arm, and the auto-merge request waited about three hours with nothing reporting why until the checks were read by hand. That is the cost the CI-only enforcement carries: the check runs after the forge has kept the text, and a landing that waits on auto-merge does not surface it.
