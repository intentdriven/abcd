---
schema_version: 1
id: "iss-2611150350117762"
slug: "a-session-renamed-because-another"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "review of the session-name hold fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "Tell a title the person chose from a title the host never updated: keep, on the session's given-names record, whether a prompt has ever shown the new name. While none has, a prompt (or a resume) that still carries the replaced title is sent the new name and the notice again, at most once per prompt and without a fixed count; once a prompt has shown the new name, a return to the old title is the person's choice and is left alone. Proven by register tests: rename output dropped, process ends, resume with the old title, next prompt delivers the new name and one notice; two dropped outputs in a row are followed by a third sending; a person who saw the new name and took the old one back is sent nothing."
---

A session renamed because another running session holds its name can end up sharing that name when the rename's output never reaches it. Two forms. First: the prompt that renames it has its hook output dropped, the session's process ends before its next prompt, its register entry is dropped, and the session is resumed still carrying the old title; its own record now names the new name, so the old title is read as one it may not hold and nothing more is sent, and it keeps the old title beside the session that holds it. Second: the rename's output and its one re-send are both dropped; the re-send is bounded at one so that a person who takes the old title back is not fought, and after it the old title is left alone. Found in review of the fix for iss-2611141504101439, which re-sends a dropped rename once while the session keeps running.
