---
schema_version: 1
id: "iss-2610262037119222"
slug: "the-only-glossary-entry-for-session-is"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 705 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/interview/session.md:29"
remedy: "none (filed automatically)"
finding_ref: cfk-179ad29f7ffae124
---

The only glossary entry for session is the unshipped grill exchange, which it says not to use generically; the implement chapter's session is a host agent session.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/interview/session.md:29` — "Do not import it into HTTP/auth contexts or use it generically"
- End B: `.abcd/development/brief/04-surfaces/27-implement.md:71` — "A session joins under its id, in the role `first` or `second`, optionally stating an agent ceiling."

session.md defines a session as one grill interview exchange, a design target, and says not to import the term or use it generically. 27-implement.md uses session throughout for a host agent session that joins a shared run in a role, a concept with no glossary entry. The same word names two concepts, and the sense the record operates on is the unregistered one.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 705.
