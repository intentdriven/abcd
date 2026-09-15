---
schema_version: 1
id: "iss-2608300320167335"
slug: "itd-183-third-round-residue"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-183 third-round security review, 2026-08-30"
found_at: "internal/core/reading/project.go, internal/surface/cli/reading.go, internal/core/reading/assemble.go"
resolution: "Four residues. A heading that renders as an excluded title but differs in bytes — bold, a code span, a non-breaking space — is now compared by the site's own anchor slug, which is the equivalence 'renders as the same heading' actually needs and one function rather than a table of markup shapes to keep current. A frontmatter block whose keys carry leading indent is valid YAML the field reader looks past, so the key pattern matches the indent, fail-closed. A relative --out is still resolved against the working directory for the core, but the operator's own string goes back on the result, because an absolute path nobody typed on the success surface is a local path leaving the machine the moment the plugin page's report instruction is followed. And the parse-fallback comment now names its actual reach: a top-level tag in a .json file, escaped."
impact: internal
---

itd-183 third-round residue: headings that render identically to an excluded title but differ in bytes (bold, code span, an NBSP inside) travel — compare slugs in the verifier and refuse; a relative --out the operator typed is absolutised against the working directory and echoed unscrubbed on the success surface and in --json out_dir, which the plugin page tells the host to report — echo the operator's string or route through the root redaction; a frontmatter block whose keys are all indented one space is valid YAML that both the field reader and the excluded-key pattern miss so origin travels — refuse; the parse fallback comment overclaims (an escaped tag in a yaml or toml file or a JSON array is admitted; hand-rewritten, so ruling 18 holds).
