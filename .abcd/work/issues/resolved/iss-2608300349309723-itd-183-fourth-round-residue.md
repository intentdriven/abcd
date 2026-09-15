---
schema_version: 1
id: "iss-2608300349309723"
slug: "itd-183-fourth-round-residue"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-183 fourth-round security review, 2026-08-30"
found_at: "internal/core/reading/project.go (sameRendering, excludedKeyInFirstBlock), internal/core/reading/assemble.go (--out refusals), internal/core/reading/status.go"
resolution: "Six residues. A raw HTML heading is refused, since the site's page reader refuses a raw HTML block rather than admitting one, so a heading spelled that way is a heading to every other reader and nothing to the markdown scan. The render comparison now strips HTML tags and comments, unwraps link labels and decodes the entities that change whether two titles look alike, before slugging. Three YAML key spellings the field reader does not report are refused: the explicit-key form, a flow mapping at top level or nested, and a double-quoted key containing a backslash — refused on the escape itself rather than on the name it hides, because resolving that name means a YAML decoder this package will not grow. The two --out refusals quote the operator's own string, which the root redaction cannot recover once the path is resolved from a subdirectory. The indent bound is spaces only, since a tab makes a code block. And the charter states that the bare verb reads run-directory names under this instrument's own scratch and opens nothing in the local tier. A heading nested inside a blockquote or a list item, and a title that reaches the excluded one through a homoglyph or an invisible format character, are both outside what this floor sees and are disclosed residue rather than caught."
impact: internal
---

itd-183 fourth-round residue, all Low: sameRendering models only emphasis and code marks, so a heading carrying an HTML comment or tag, an entity, a link wrapper, a homoglyph, or nested in a blockquote or list item renders as the excluded title but slugs differently (strip tags and comments, decode common entities, unwrap links before slugging, or refuse on a slug substring match and correct the comment); top-level YAML spellings of an excluded key that the line pattern misses travel (explicit key, top-level or nested flow mapping, a double-quoted key with an escape); the two --out refusals interpolate the resolved absolute directory built from the operator's relative string, which scrubPaths cannot redact when cwd is not a prefix; status.go reads run-directory names under the assembler's own scratch, worth a charter sentence so it is not mistaken for an invariant-14 breach.
