---
schema_version: 1
id: "iss-2610241519088123"
slug: "the-host-still-cancels-abcd-s-sessionend"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "a session exit reported by the user, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
remedy: "Stage by session id with no repository lookup at exit (the person's choice, 2026-10-24): have hook session-end run no git and no ahoy.Detect, and stage the raw transcript into one machine-level staging area keyed by session id, with the payload's cwd and transcript path beside it; have the next SessionStart place each staged transcript in its store (from the session marker NoteSessionRepo wrote, else from the staged cwd), redact and store it through the existing fail-closed capture, and name one it cannot place; correct the comment at internal/surface/cli/hook_subagent.go:95-110. Prove it with a test that hook session-end starts no git process and stages a transcript whose cwd no longer resolves, and a test that the next start files it in the right store. Grounds: the host's hooks reference (https://code.claude.com/docs/en/hooks): SessionEnd hooks have a default timeout of 1.5 seconds, and timeouts set on plugin-provided hooks do not raise the budget."
---

The host still cancels abcd's SessionEnd hook on a loaded machine, so the session's transcript is neither staged nor stored, and nothing says so. On 2026-10-24, at a load average of about 21 on 10 cores, session ad158fbf ended with 'SessionEnd hook [sh .../hooks/session-end.sh] failed: Hook cancelled' under the 0.15.1 plugin, whose hook is identical to the current source; its 559 KB transcript reached neither the staging area nor the store, though the host's own copy is still on disk. The host gives a SessionEnd hook 1.5 seconds in all, and a timeout set in a plugin's hooks.json does not raise it (the host's hooks reference, read 2026-10-24). Before it stages anything, hook session-end detects the repository through ahoy.Detect and git, with up to two retries inside a 1.2-second budget; under this load starting the binary alone takes about 0.36 s and each git call about 0.15 s. The August fix (iss-2608230817034768) took redaction off the exit path but left this lookup on it, and the comment at internal/surface/cli/hook_subagent.go:95-110 says the hook could ask the host for more time, which the reference says a plugin hook cannot.
