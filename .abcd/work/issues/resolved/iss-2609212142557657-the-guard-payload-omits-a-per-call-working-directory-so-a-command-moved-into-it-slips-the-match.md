---
schema_version: 1
id: "iss-2609212142557657"
slug: "the-guard-payload-omits-a-per-call-working-directory-so-a-command-moved-into-it-slips-the-match"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "abcd lab 2 (lab-260831162806-976575f), filed from the capstone handoff on 2026-09-21 after verification at main"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard.go (guardHookInput reads cwd and tool_input.command only)"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (the older owed ruling on the guard's working directory: extend the guard contract (option A) to read a host-supplied per-call working directory, after probing whether the host fails the call or falls back to the session directory when none is supplied; a build lane owed, not holding the tag)."
resolution: "The guard hook reads tool_input.workdir, resolves it against the session directory, and checks the command against the registry of the repository it runs in as well as the session's, keeping the stricter verdict; a malformed workdir is refused with the blocking status. Probe of opencode 1.18.31 (bash tool driven directly): a missing or non-directory workdir fails the call and nothing runs, so the workdir is never read as a cd. Option A as ruled by the product thinker on 2026-09-23. No in-tree adapter sends tool_input.workdir yet (the shipped hook's Bash tool input has no such field, and the opencode adapter is lab-only), so the path is reachable only by a host that does."
impact: additive
resolved_by:
  commit: "a02d6a1b"
---

The guard's hook payload omits a per-call working directory, so a command moved into it slips the command-string match. guardHookInput reads the session cwd and tool_input.command and nothing else; a host whose shell tool takes a per-call working-directory field (opencode's does; a future Claude Code field would) can carry a destructive command whose target is the directory rather than an argument, and the match over argv sees nothing hazardous. Live-demonstrated in lab 2's smoke (probe p5b-live-session-retest under the lab home): the model set the tool's workdir and ran the bare destructive verb. Verified at main on 2026-09-21: the struct is unchanged. Decision owed to the product thinker: extend the guard contract to read a working-directory field where the host supplies one and resolve the command against it, or document the reach in adr-42's mistake-filter scope as outside the boundary. Not a security boundary by adr-42's own statement, which is why the severity is minor; the category is security because the slip is the shape a mistake filter exists to catch.

## Grounds

- pursued: a host-supplied workdir changes the verdict only through the registry of the directory it resolves into, and never through the failed-cd entry; a host that runs the command in the session directory when the workdir is missing would show it wrong, and would need its adapter to fold the workdir into the command as a cd
