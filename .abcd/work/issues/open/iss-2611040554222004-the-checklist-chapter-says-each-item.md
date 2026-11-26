---
schema_version: 1
id: "iss-2611040554222004"
slug: "the-checklist-chapter-says-each-item"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/40-checklist.md"
remedy: "Give checklist add and set a repeatable --owner-ref flag that checks the branch:, pr: and record-id shapes, and a credential flag naming its kind, scope and expiry, with a test that the stop hook's behind-note clears once an item names the branch; or mark both fields in the chapter and the page as schema only until a verb writes them."
---

The checklist chapter says each item carries the references it is about and an optional credential named by kind, scope and expiry (.abcd/development/brief/04-surfaces/40-checklist.md:35), and commands/checklist.md:77-78 says an item needing a credential names it that way. The schema validates and the list renders owner_refs and credential (internal/core/checklist/checklist.go:138-139), but `abcd checklist add` has only --next-look, --promise and --waits-on, `set` has only --state, and no non-test code assigns either field. Neither can reach an item through a shipped write.

The gap has a consequence a session meets at the end of a turn. The stop hook's note that the list is behind (internal/core/autonomy/checklist_behind.go:110) and commands/checklist.md:105-108 tell a session to name each of its branches and reservations in an item's owner_refs, using `abcd checklist add` or `abcd checklist set`. Neither verb takes a flag that writes owner_refs (internal/surface/cli/checklist.go:136-163), so the note cannot be cleared through the verbs.
