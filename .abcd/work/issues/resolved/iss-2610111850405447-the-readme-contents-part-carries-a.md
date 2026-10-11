---
schema_version: 1
id: "iss-2610111850405447"
slug: "the-readme-contents-part-carries-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "review of README parts step 2, run-2610110908176735"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/readme_derive.go"
remedy: "Strip inline HTML tags from a heading's text before it is listed and before its anchor is slugged."
resolution: "Inline HTML tags and comments in a heading are dropped from the contents entry's text and from its anchor, keeping the space a dropped tag leaves in the anchor as the forge does."
impact: fix
resolved_by:
  commit: "7d9a396381657fb202a45142e8a66b75ce7ecdc1"
---

The README contents part carries a heading's inline HTML raw into the list text and its tag characters into the anchor: `## <a name="install"></a> Install` is listed with the tag in its link text and an anchor built from the tag's characters, where the forge renders the heading as `Install` with the anchor `#-install`.

## Grounds

- pursued: a heading carrying inline HTML is listed by its visible text and linked to the forge's anchor; a tag in a contents entry or an anchor built from tag characters shows it wrong
