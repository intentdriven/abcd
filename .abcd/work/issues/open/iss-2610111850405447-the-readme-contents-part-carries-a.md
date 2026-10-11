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
---

The README contents part carries a heading's inline HTML raw into the list text and its tag characters into the anchor: '## <a name="install"></a> Install' is listed with the tag in its link text and an anchor built from the tag's characters, where the forge renders the heading as 'Install' with the anchor #-install.
