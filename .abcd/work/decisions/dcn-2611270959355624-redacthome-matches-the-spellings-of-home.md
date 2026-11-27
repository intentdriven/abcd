---
id: dcn-2611270959355624
slug: redacthome-matches-the-spellings-of-home
date: 2026-11-27
---
2026-11-27 — RedactHome matches the spellings of HOME, not every spelling a text could give the same directory. A path printed with a doubled slash or a dot segment under a clean HOME (for example /Users//alice/notes under HOME [redacted-path]) is left whole, because text spellings are unbounded and matching them all would mean normalising the text, which risks rewriting content that is not a path. The generic /Users and /home detectors and the username sweep remain the backstop for the stores. To reverse it, clean each path-shaped token of the text before matching, with tests that text which is not a path is left untouched. Tracked by iss-2609290055092630.
