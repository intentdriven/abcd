---
schema_version: 1
id: "iss-2609020734198139"
slug: "the-v0-7-1-brief-surface-crosscheck-direction-a-tier-full-ov"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "release-v0.7.1-crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
---

The v0.7.1 brief-surface crosscheck (Direction A, tier full, over the sixteen first surface chapters) found four false claims in the design record, none in a user-facing page. 01-ahoy.md line 225 says every non-SessionStart shim attempts hooks/bootstrap.sh when the plugin-root binary is missing; hooks.json wires bootstrap into UserPromptSubmit, SessionStart, PreToolUse and PreCompact but not SessionEnd, which resolves the plugin root, then a vouched PATH abcd, then fails loudly, so a machine with no plugin-root binary never stages its transcript at exit and the chapter says otherwise. 01-ahoy.md line 28 lists status among the sub-verbs that ship on the CLI; abcd ahoy status exits 2, status being a plugin-side alias for the bare render, and the chapter's own sub-verb table omits it. 07-memory.md line 3 disambiguates spc-38 and spc-39 by saying the store's ids stop well short of spc-38; the store now runs to spc-70 and both ids name unrelated specs, so a dozen bare references on the page point at the wrong records. 07-memory.md line 116 enumerates launch --dry-run's gates as five, omitting semantic-receipts, which 04-launch.md documents. Each is a one-line correction; the receipt for the v0.7.1 cut defers them to this record.
