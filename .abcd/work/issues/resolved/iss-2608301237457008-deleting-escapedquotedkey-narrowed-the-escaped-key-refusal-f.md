---
schema_version: 1
id: "iss-2608301237457008"
slug: "deleting-escapedquotedkey-narrowed-the-escaped-key-refusal-f"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "itd-183-round-9-security"
found_at: "internal/core/reading/project.go"
resolution: "The line-level escaped-key refusal is restored alongside the positional scanner's, and the scanner's blank skip covers YAML's whitespace rather than space and tab alone. The narrowing the deletion rested on is now disclosed where it happens."
impact: fix
resolved_by:
  intent: "itd-183"
---

deleting escapedQuotedKey narrowed the escaped-key refusal from regex whitespace to space and tab so a CR-preceded escaped excluded key is admitted

Found by the round-9 adversarial security review of build/itd-183.
REGRESSION INTRODUCED BY 2225d6cb. This is the answer to the question the
orchestrator put to both reviewers: the deleted check was NOT strictly
subsumed.

The deleted `doubleQuotedKeyRe = ^\s*"([^"]*)"\s*:` used Go's `\s`, which is
[\t\n\f\r ]. The positional scanner's `skipBlanks` covers only space and tab.
So a frontmatter line whose bytes are `CR "origin": <value>` is a
top-level `origin` key to a YAML reader (the CR is a line break), but
`blankQuoted` hits `\r` in its default arm, sets scalar = false, never opens
the quoted token, and so never reports it in `quotedKeys` -- meaning the
`strings.Contains(name, "\\")` refusal at project.go:872 is never reached.
`excludedKeyLineRe`'s quoted alternative rejects the `\` in the name. The value
travels.

The mirror shape `"origin"\r: <value>` fails the same way at the
skipBlanks colon test.

Verified: HEAD admits, parent refuses with `still carries the excluded key
"ori\\u0067in" at line 3`. Only ESCAPED names slip -- the un-escaped `CR
"origin":` still refuses via excludedKeyLineRe.

Remedy: make `skipBlanks` skip \r (and \f) as well as space and tab, or re-add
the escaped-key refusal as a line-level check using `\s*`. A CR inside a
frontmatter line is never legitimate content in this corpus.

LESSON, worth carrying: the round deleted this check as an orphan on the claim
that the new scanner reaches "strictly more" spellings. It reaches more of
some, fewer of others. A deletion justified by subsumption needs the
subsumption proved, not asserted.
