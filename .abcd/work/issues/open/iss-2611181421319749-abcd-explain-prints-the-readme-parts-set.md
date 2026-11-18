---
schema_version: 1
id: "iss-2611181421319749"
slug: "abcd-explain-prints-the-readme-parts-set"
severity: "nitpick"
category: "observation"
source: "agent-observation"
found_during: "review of the list-valued flags change"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/explain.go"
remedy: "give the list facts a way to name a member that stands alone (for example an exclusive list beside allowed), have enumListFlag record it from the parser's own none constant, and print it in explain as a separate alternative, such as each one of release-badge | ... | contents, or none alone"
---

abcd explain prints the readme-parts set of ahoy install as each one of release-badge | build-badge | abcd-badge | licence | links | contents | none, which reads as though none may be listed beside a part, while the parser accepts none only on its own and refuses licence,none at exit 2 before anything is written; the facts carry no way to say that one member of a list set stands alone
