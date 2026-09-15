---
schema_version: 1
id: "iss-2609020543544634"
slug: "a-here-document-delimiter-that-is-not-letter-led-fell-to-the"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "isDelimStart now reads the conservative superset of the delimiter words bash accepts — a letter, a digit, an underscore, or a dollar-led word — so cat <<20, cat <<$D and the <<20 of let mask=1<<20 open here-documents instead of falling to the arithmetic-shift branch, where the document was tokenized as commands and one apostrophe raised ErrUnparsableCommand for the hook to fail open on. A dollar-led delimiter is a word whose value the guard cannot know, so the document never terminates and the existing fail-closed heredoc-unterminated block answers rather than an error. The enclosing arithmetic context (inArithmetic, checked before this) stays the sole shift discriminator. Proved by TestNonAlphabeticHeredocDelimiterOpensADocument (internal/core/guard), seven rows, watched failing first on a scratch archive of the pre-fix HEAD where four rows returned exit 2 or a silent allow; the three controls — an arithmetic shift, a parenthesised arithmetic shift, and a numeric-delimited document whose body is only data — passed before and after. The residual left out of the set (a delimiter bash allows that this set does not, such as <<! or a path, and bash's unmodelled $[ … ] arithmetic form) is named in the guard chapter and carried as iss-2609020544031234."
impact: fix
---

A here-document delimiter that is not letter-led fell to the arithmetic-shift branch, so the DOCUMENT was tokenized as commands and one apostrophe in the body raised ErrUnparsableCommand — which the pre-tool-use hook maps to fail-OPEN, leaving any hazard on a later line to run unguarded. bash accepts a WORD after <<: 'cat <<20', 'cat <<$D' and the '<<20' of 'let mask=1<<20' are all here-documents (the shell tokenizes the redirection before the let builtin sees arithmetic), and each of those spellings reached exit 2 / NOT CHECKED on v0.7.0. Evidence symbol: isDelimStart (internal/core/guard/tokenize.go), consulted at the '<<' branch of tokenize. It also falsified the claim both front doors make — that a quote inside a here-document body is document text and never unparsable. The fix must read the conservative superset of delimiter words (letter, digit, underscore, $-led) so the body is data, with the enclosing arithmetic context left as the sole shift discriminator.

## Grounds

- pursued: the failing direction was fail-OPEN on the hook, which is the one direction adr-42 rules out, and the delimiter word is a bash fact rather than a design choice — bash tokenizes the redirection before any builtin sees arithmetic, which is why let mask=1<<20 is a here-document there too. Taking the conservative superset rather than every word bash accepts keeps a << reached in an unmodelled arithmetic context from swallowing the rest of the input on the strength of an operator.
