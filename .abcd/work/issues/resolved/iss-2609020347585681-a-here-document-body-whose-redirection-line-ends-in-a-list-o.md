---
schema_version: 1
id: "iss-2609020347585681"
slug: "a-here-document-body-whose-redirection-line-ends-in-a-list-o"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "A pending here-document body is now consumed at the newline that ends the redirection line, whatever that line ends in, so a trailing list operator no longer defers it and document text never reaches command position; the remaining ErrUnparsableCommand class is named exactly, in errors.go, commands/guard.md, both cobra Long texts and the guard chapter, as an unterminated quote in COMMAND text, with a quote inside a body called out as document text. Proved by TestHeredocBodyStartsDespiteTrailingListOperator (internal/core/guard), watched failing first on a scratch archive of the pre-fix tree, where the apostrophe row returned ErrUnparsableCommand; the tokenize_test case that pinned the opposite reading is inverted with the bash 3.2 probe that settles it recorded beside it."
impact: fix
---

A here-document body whose redirection line ends in a list operator reaches command position instead of being read as data, so an apostrophe in an ordinary document becomes ErrUnparsableCommand and the pre-tool-use hook fails OPEN and runs the whole line unguarded (CWE-636, the GHSA-5wx3-2c86-fjpx fail-open class, pre-existing at v0.7.0). Evidence symbol: tokenize's newline branch in internal/core/guard/tokenize.go, which withheld skipHeredocBodies while lastList was set. bash collects here-document bodies at the end of the PHYSICAL line, not at the end of the command list: probed on bash 3.2, `cat <<EOF &&` / `don't` / `EOF` / `echo ok` runs with the body as data and prints ok. The fix must start the body on the line after the redirection regardless of a trailing `&&`, `||` or `|`, and must leave ErrUnparsableCommand for an unterminated quote in COMMAND text only, which no shell runs either.

## Grounds

- pursued: bash collects here-document bodies at the end of the physical line, so the guard reads the body where bash reads it; this would be shown wrong if some shell an agent runs under deferred a body past a trailing list operator, in which case the deferral was right and the body really is command text
