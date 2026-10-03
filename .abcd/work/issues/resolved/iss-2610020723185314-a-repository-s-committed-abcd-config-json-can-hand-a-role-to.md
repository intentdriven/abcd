---
schema_version: 1
id: "iss-2610020723185314"
slug: "a-repository-s-committed-abcd-config-json-can-hand-a-role-to"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
remedy: "Rulings RN2, OC1 and OC2 (2026-10-02): only a personal route (the machine's ~/.abcd/config.json, or the person's own invocation) may hand a role to a runner; a repository route to a runner is skipped with a diagnostic naming the file, the key and the machine's file, and the role runs on the host as if unrouted (the claude runner keeps --print --bare on an API key; code.claude.com/docs/en/cli-reference and code.claude.com/docs/en/legal-and-compliance, read 2026-10-02). The opencode runner launches with OPENCODE_DISABLE_PROJECT_CONFIG=1, OPENCODE_DISABLE_CLAUDE_CODE=1 and OPENCODE_DISABLE_EXTERNAL_SKILLS=1 beside --pure (opencode's source flag.ts, config.ts, instruction.ts, read 2026-10-02), held by a hostile-fixture test asserting the exact argv and env."
resolution: "Only a personal route hands a role to a runner: a repository route to a runner is skipped with a diagnostic and the role runs on the host; the opencode runner launches with its project-configuration, CLAUDE.md and external-skills switches beside --pure, held by a hostile-fixture test on the exact argv and env (rulings RN2, OC1, OC2 of 2026-10-02)."
impact: breaking
resolved_by:
  commit: "81ab91072"
---

A repository's committed .abcd/config.json can hand a role to a command-line runner (roles.<role>.runner: claude or opencode), so a checkout decides when the person's paid key is spent on its behalf; and the opencode runner is sealed only against plugins (--pure), so the repository's AGENTS.md, CLAUDE.md, opencode.json, project agents and skills still reach the run.

## Grounds

- pursued: we expect no repository's committed configuration to start a runner on the person's key, and no repository instruction, setting, agent or skill to reach an opencode run; shown wrong if a repo-layer roles.<role>.runner naming claude or opencode yields a runner route, or if the opencode launch lacks any of the three switches or --pure
