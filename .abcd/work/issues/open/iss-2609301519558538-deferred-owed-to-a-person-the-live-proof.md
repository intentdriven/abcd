---
schema_version: 1
id: "iss-2609301519558538"
slug: "deferred-owed-to-a-person-the-live-proof"
severity: "major"
category: "future-work-seed"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner"
remedy: "Waits on a ruling: either the product thinker rules the runner may close on fakes with the live proof recorded as owed (as ruling L does for the three paid adapters), and spec close runs with Delivers, or a person runs checks (1) to (3) with their own credential and records each outcome on this record before the close."
---

Deferred, owed to a person: the live proof of the command-line runner (itd-2609201916056194). Phase 1 and 2 prove every criterion on fake harness binaries only; no ruling admits a live check of the runner (ruling L names the site setup, the API adapter and the decision adapter; H8 lists the three paid-service checks and itd-6's RepoPrompt run), so the intent cannot close. What a person must run: (1) the intent's first proof, criterion 7 live: a ruthless review of a real lane routed roles.ruthless-reviewer.runner=opencode, from a lane the host drives, its return recorded beside a host-run review's and differing only in the route; (2) the claude runner under --bare with --permission-mode dontAsk and --allowedTools, confirming a nested claude does not refuse under an inherited CLAUDECODE, and noting that the headless page (read 2026-09-30) says bare mode never reads OAuth or the keychain, so the claude runner needs ANTHROPIC_API_KEY, a paid API key rather than the person's subscription; (3) opencode run --format json --pure: the event shape (the CLI page does not document it; the adapter assumes step_start/text/step_finish with a sessionID and a part) and whether run mode prompts for a permission, which the timeout bounds and records as a fallback.

2026-10-02 — Added by the sec-runnerSeal review (lane runnerSeal), a fourth live check, medium in the reviewer's grading and unverified: (4) whether opencode reads a `.env` from its working directory. The runner launches opencode with the repository (the lane's worktree) as its working directory (internal/core/runner/proc.go), and opencode is a Bun-built binary; if its runtime auto-loads a `.env` from the working directory, a repository's committed `.env` sets names the seal does not override, OPENCODE_CONFIG, OPENCODE_CONFIG_CONTENT or a provider base-URL variable, which reach opencode's configuration or redirect the person's key past the three seal switches (OPENCODE_DISABLE_PROJECT_CONFIG, OPENCODE_DISABLE_CLAUDE_CODE, OPENCODE_DISABLE_EXTERNAL_SKILLS) and --pure. The probe: plant a `.env` in a scratch repository naming OPENCODE_CONFIG at a file the repository holds, run the opencode runner there, and observe whether opencode reads that file. If it does, launch opencode from a directory the repository does not control, or clear those names in the launch environment; neither is applied without the check, because both change what the person's own environment hands opencode. Severity is raised from minor to major to match the reviewer's medium.
