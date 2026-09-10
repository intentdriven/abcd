---
schema_version: 1
id: "iss-2609100505142469"
slug: "capture-redactor-corrupts-reverse-dns-identifiers"
severity: "critical"
category: "bug"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-07/08; re-filed into abcd 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (capture, scanner redaction)"
---

The capture redactor rewrites the leading component of a reverse-DNS identifier when it happens to equal the local account name, silently corrupting technical content in a permanent record. This refines iss-2609061504302157, which reports the same root cause on an ordinary dictionary word; the dotted-identifier class is broader, and its damage is unrecoverable rather than merely noisy.

Observed capturing a bug whose whole hypothesis turned on a platform bundle identifier. The text named the identifier as `<prefix>.<product>.app`, where the prefix is the ordinary reverse-DNS-style first component. The machine's account name is that same short word. The redactor matched it as a username and the record was written with `[redacted-user].<product>.app`. The identifier is the thing the issue is about, and it is now unrecoverable from the record: a reader cannot tell which word was replaced, and the capture is the only place the hypothesis was written down.

Three things make this worse than a false positive.

It is silent at the point it matters. `capture --json` returns `redacted: 1` and nothing else: not the span, not the before, not the rule that fired. The plugin surface is told to relay the count, so a caller can say "one span was rewritten" and still not know what changed. A count is not a diff. Finding the corruption required grepping the written record and comparing it to the input by eye.

The collision class is large and ordinary, not exotic. Reverse-DNS identifiers begin with a small set of short words: `com`, `io`, `app`, `net`, `org`, `me`, `sh`, and the usual three-letter abbreviation for development. Every one of them is a plausible Unix account name. Any repo whose maintainer's account name is one of those words cannot write its own bundle identifier, module path, package name or domain into a capture without it being mangled. The likelihood is not the maintainer being careless; it is two very short common words coinciding.

It corrupts rather than refuses. Everywhere else abcd fails closed and says so: a malformed banlist line refuses the commit, an unreadable identity pin blocks, a missing docs root refuses the lint rather than passing. Here the write succeeds, the record looks clean, and the damage is only visible to someone who still has the input. That is the false-green shape the loud-staging principle exists to forbid, applied to the record store itself.

Needed, roughly in order. Report what was redacted, not how many: the JSON should carry each span's rule and its position so a caller can show the user the change and undo it. Do not match a bare account name inside a dotted identifier; a token bounded by dots on both sides, or followed by a dot and a known TLD-shaped component, is a namespace, not a home directory, and the existing rule already knows how to recognise a path, which is the shape that actually leaks. Offer an opt-out for a span the author asserts is safe, the way `abcd-lint:allow` works for the privacy rule, so a maintainer whose account name is a reverse-DNS prefix can still write their own identifier.

Related: the `privacy-hygiene` lint rule flags persona-derived paths the conventions mandate, filed separately. Both are the same underlying problem, identifier-shaped text judged by a rule that only models personal identifiers.
