---
schema_version: 1
id: "iss-2610111701536641"
slug: "the-status-line-should-show-which"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "orchestrator session, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusline/payload.go"
remedy: "Have abcd's own SessionStart hook name the session through the host's documented hookSpecificOutput sessionTitle (code.claude.com/docs/en/hooks, checked 2026-10-11), never by touching ~/.claude, and only when the person has not named it (--name or /rename); the name is the next free persona name from the repository's persona registry (.abcd/development/personas.json, else the bundled Alice, Bob, Carol, Dave sequence), as the product thinker proposed on 2026-10-11 because a familiar name is easier to find than a code and the chance of reading it as a role is manageable; the status line then shows session_name from its payload. Prove it live: a fresh session shows its persona name in the status line and in a peer's session listing, two concurrent sessions get different names, a hand-named session keeps its name, and a resume or compaction does not rename it."
related_intents: [itd-2610111707085098]
---

The status line should show which session it belongs to, the short name peer sessions use to message each other (for example abcd-60 or abcd-a5), so a person running several sessions in one repository can tell them apart and address one. Checked against the host's status-line documentation (code.claude.com/docs/en/statusline, 2026-10-11): the payload abcd's statusline verb reads on stdin carries session_id (a UUID) and session_name, which is present only when the person named the session (--name or /rename) or the host generated a title, but never that default short name, and no environment variable or file exposes it. The short name cannot be derived from session_id: this session is abcd-0b while its id begins d3616efc. internal/core/statusline/payload.go reads neither field today. Related: iss-2609291032194222, whose session-bound remedy also reads the payload's session id.

## Grounds

- pursued: We expect a familiar persona name to make a session easier to find and address than a generated code. Shown wrong if people still mix sessions up, or keep reading the name as a role.
