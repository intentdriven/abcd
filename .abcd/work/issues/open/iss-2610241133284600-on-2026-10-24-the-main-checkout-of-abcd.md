---
schema_version: 1
id: "iss-2610241133284600"
slug: "on-2026-10-24-the-main-checkout-of-abcd"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "pushing design/autonomy-grant"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-push"
remedy: "Have every abcd verb that commits or pushes (capture, decide, build, implement, land, drain) and the SessionStart hook refuse, or at least say loudly, when the checkout's core.hooksPath does not resolve to the committed .githooks, naming abcd ahoy install as the fix; and add the check to abcd ahoy's gap list."
---

On 2026-10-24 the main checkout of abcd at ~/Confidential/ABCDevelopment/abcd had no git hooks installed: core.hooksPath was unset in every config scope and the common .git/hooks held only samples, so the committed .githooks (commit-msg, pre-commit, pre-push) ran for no commit and no push made from it or from its linked worktrees. Several sessions were committing and pushing that day. Nothing said so: commits went through silently, a commit message naming an issue without a Refs: line (RS004) was accepted at commit time, and a push to origin was made with no preflight receipt, which the pre-push hook exists to refuse. The gap surfaced only because make preflight's lint-issues gate failed afterwards. How the setting was lost is not known; the setup redesign under way the same day touches the hooks. abcd ahoy reports install state but did not flag it, and no commit verb checks it before writing.
