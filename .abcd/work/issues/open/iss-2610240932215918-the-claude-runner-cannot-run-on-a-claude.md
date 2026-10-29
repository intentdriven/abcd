---
schema_version: 1
id: "iss-2610240932215918"
slug: "the-claude-runner-cannot-run-on-a-claude"
severity: "major"
category: "architectural-insight"
source: "agent-finding"
found_during: "/abcd:build iss-2610140652230444 session, 2026-10-24 (intent interview refused on a subscription account)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/claude.go"
remedy: "Waits on a ruling of the technical facilitator reversing RN2 (dcn-2610020000000016) for a route with no API key, asked only after a live probe shows the isolation flag set seals CLAUDE.md, skills, agents, plugins, hooks and MCP as --bare does (read from the run's system/init event). Then: when the route carries no API key (the variable's presence, never its value, or a machine setting naming the credential), start the person's own claude -p without --bare under that flag set and refuse the run when system/init shows anything loaded outside it; never read, mint (setup-token) or store a token; keep --bare for a route with a key; and say on stderr which credential kind the run used."
deferred_after: v0.15.1
deferral_reason: "Waits on the technical facilitator's ruling on reversing RN2 (dcn-2610020000000016) after a live isolation probe; subscription owners keep running roles in their own session, as v0.15.1 documents (ruled 27 October 2026)."
---

The claude runner cannot run on a Claude subscription, so a person without an API key has no route to any role abcd runs from a plain terminal: the planning interview, a build's implementer and validators, the drain's judge.

`internal/core/runner/claude.go:116-157` always starts `claude --print --bare …`. The vendor's headless documentation (https://code.claude.com/docs/en/headless and https://code.claude.com/docs/en/authentication, read 2026-10-24) says `--bare` never reads OAuth credentials or the keychain and ignores `CLAUDE_CODE_OAUTH_TOKEN`, so on a subscription-only login every routed role fails at launch, with no early check naming why. Surfaced 2026-10-24 when `abcd intent interview` refused a person whose account is a subscription and the only remedy it offered was a runner they cannot run.

The restriction is ruling RN2 of the technical facilitator (dcn-2610020000000016, 2026-10-02: "runner keeps --bare with an API key … subscription-only owners run the role in their own session"), restated in the brief (`.abcd/development/brief/04-surfaces/34-build.md:465-470`, "The claude runner stays bare, on an API key alone") and built on by dcn-2610161624025493, whose headless shell approver is designed around `--bare` ("dropping --bare would let the repository's hooks, plugins and CLAUDE.md run"). RN2 rests on two grounds, and both have to fall for the remedy to stand:

1. **Isolation.** RN2's research (the cli-reference, read 2026-10-02) found that "no flag set short of the bare flag seals project instructions, plugins, skills and agents". The documented flags read on 2026-10-24 (https://code.claude.com/docs/en/cli-reference) cover settings sources, MCP servers (`--strict-mcp-config --mcp-config`), inline `--settings`, tools (`--tools`/`--allowedTools`) and plugins (`--plugin-dir`), but no page read states that any of them stops CLAUDE.md, skills, agents or auto-memory loading. This ground is not shown to fall: it has to be proven by a live probe that starts `claude -p` without `--bare` under the candidate flag set in a repository carrying a CLAUDE.md, a skill, an agent, a plugin and a hook, and reads the run's `system/init` event for what loaded, before anyone is asked to rule.
2. **Terms.** The vendor's legal page as read on 2026-10-24 (https://code.claude.com/docs/en/legal-and-compliance) bars a developer from routing requests through Free, Pro or Max credentials on behalf of their users and from collecting or storing Claude.ai tokens, and states that it does not prevent an end user from signing in to the unmodified Claude Code binary with their own subscription; Pro and Max limits assume ordinary individual use. On that reading, abcd starting the person's own `claude` on their own machine under their own login is the permitted case. A second reviewer could not fetch the page, so the reading is one session's and is to be re-read when the ruling is asked.

Reversal flag (advisory, for the technical facilitator to rule): the remedy reverses RN2 for a route with no API key, and the brief paragraph and the `--bare` premise of dcn-2610161624025493 with it; dcn-2610161624025493's approver, which reaches the run through `--mcp-config`, is unaffected by the flag itself. Nothing is applied until the probe has run and the ruling is given.

The runner has no notion of a key today (no `ANTHROPIC_API_KEY` reference in `internal/core/runner/` outside tests), so "a route with no API key" needs a stated test: the presence of the variable in the runner's environment, never its value, or an explicit machine setting naming the credential the route uses.

## Probe 2026-10-24: the isolation ground, tested live

Claude Code 2.1.310 on macOS, a subscription login with `ANTHROPIC_API_KEY` and `CLAUDE_CODE_OAUTH_TOKEN` unset, model `haiku`, in a throwaway repository under `~/.abcd.noindex/probes/<root-sha>/safe-mode-2610241000/` carrying a decoy `CLAUDE.md` (an instruction to say a fruit word), a project skill, a project agent, a project `SessionStart` hook and a project `.mcp.json`. Each run read the `system/init` event and the result.

| Flag set (beside `-p --output-format stream-json --verbose --no-session-persistence --tools Read`) | Auth | Project CLAUDE.md, skill, agent, hook, MCP | User plugins | abcd's `--mcp-config` approver | abcd's `--settings` hook | Auto-memory |
|---|---|---|---|---|---|---|
| `--safe-mode` | ran, `apiKeySource: none` | none loaded | not loaded | **not loaded** | **did not run** | off |
| `--setting-sources "" --disable-slash-commands --strict-mcp-config --mcp-config <approver> --permission-prompt-tool mcp__abcd__approve` | ran, `apiKeySource: none` | none loaded | not loaded (only the harness's built-ins) | connected | ran | **on** |
| the row above with `--settings '{"autoMemoryEnabled":false}'` | ran | none loaded | not loaded | (empty config) | — | off |

What it shows: a subscription login runs `claude -p` without `--bare`; `--safe-mode` seals the project but also drops abcd's own approver and hook, so it cannot replace `--bare` for a shell role; the second set, with auto-memory turned off in the inline settings, sealed every decoy the probe planted while keeping the approver and abcd's hook. What it does not show: the user layer's own `CLAUDE.md` and skills (the probe planted none there), output styles, and behaviour on another Claude Code version; the runner should still refuse a run whose `system/init` lists anything outside the expected set.

## Deferral 2026-10-27

Deferred past v0.15.1: Waits on the technical facilitator's ruling on reversing RN2 (dcn-2610020000000016) after a live isolation probe; subscription owners keep running roles in their own session, as v0.15.1 documents (ruled 27 October 2026).
