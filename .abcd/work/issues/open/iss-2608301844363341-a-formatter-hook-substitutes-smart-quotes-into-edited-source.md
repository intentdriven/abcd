---
schema_version: 1
id: "iss-2608301844363341"
slug: "a-formatter-hook-substitutes-smart-quotes-into-edited-source"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "itd-189-delta-builder"
found_at: "internal"
---

a formatter hook substitutes smart quotes into edited source comments which passes gofmt and every gate and is visible only in a diff

Reported by the itd-189 delta builder, which hit it and caught it. Writing
`!!str \'\'` into a Go `//` comment came back with the straight pair replaced by a
typographic quote -- inside a comment that was ABOUT YAML spellings, so the
substitution silently falsified the very thing the sentence documented.

What makes it worth a major rather than a note is the detection profile. The
result compiles, `gofmt` is silent, every gate passes, and reading the file back
looks right, because the glyphs are nearly indistinguishable at a glance. Only a
diff shows it. So the failure mode is a comment or a string that quietly stops
saying what its author wrote, in a repository whose standing defect class this
cycle is a claim asserting something that is not the case.

Scope, measured rather than assumed: `grep -rP` for the four curly glyphs across
`internal/` and `cmd/` in all four worktrees finds the SAME four files in each,
and all four uses are legitimate -- a regex matching a curly apostrophe in a
name, a rune-index helper for curly quotes, the positioning normaliser that maps
smart to straight, and a test comment using them deliberately. Nothing corrupt
is committed today.

The reach is what is unknown. It fires on an editing tool's write path, so
agent-authored edits are exposed and shell-authored ones (heredocs, `cat`) are
not. Every agent this session wrote through the editing path.

Two cheap mitigations, neither implemented: a lint rule refusing the four curly
glyphs in Go source outside an allow-list of the four files above, which would
be armed and deterministic; and a line in agent briefs to diff their own edits
before committing rather than reading them back. Same family as
iss-2608301715040589, the interactive `cp` hazard: a machine-local property that
silently breaks an otherwise correct instruction.

