---
schema_version: 1
id: "iss-2608300258519825"
slug: "heading-and-key-floor-spellings-and-fence-blind-verifier"
severity: "major"
category: "security"
source: "impl-review"
found_during: "itd-183 second-round reviews, 2026-08-30"
found_at: "internal/core/reading/project.go (redactExcluded, verifyRedaction, excludedKeyLineRe)"
resolution: "The heading half matched the title the section scanner reports, so an ATX closing sequence or a setext underline carried an excluded section past it; closing hashes are normalised away in the redactor and the verifier alike, and a setext heading is a refusal rather than a redaction because the section scan does not model one and inventing a second heading scanner to compute its span is the second parser this package exists not to grow. The key half now reads a quoted key and treats a blank line inside a block scalar as part of it. And the verifier runs over the same fence-aware section scan the redactor spans by: reading raw lines made a fenced example of the record template refuse every assembly the repository could run."
impact: fix
---

The exclusion floor's heading half matches the title the section scanner reports and neither the redactor nor the verifier normalises an ATX closing sequence or recognises a setext heading, so an excluded section travels under '## Audit Notes ##', '## Open Questions #' or an underlined heading while the manifest asserts it refused; the key half misses a quoted key ("origin":) and a block scalar with an internal blank line; and the verifier is not fence-aware while the redactor is, so a fenced example of the record template in any admitted markdown refuses every assembly. Normalise closing hashes and detect setext in both places, accept a quoted key, skip blank lines inside a block, and verify over the same fence-aware section scan the redactor uses.
