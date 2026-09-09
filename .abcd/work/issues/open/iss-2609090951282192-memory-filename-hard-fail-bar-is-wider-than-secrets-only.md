---
schema_version: 1
id: "iss-2609090951282192"
slug: "memory-filename-hard-fail-bar-is-wider-than-secrets-only"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/redact.go"
---

The memory write boundary refuses a page filename carrying a hard-fail span, and the ruling that ordered it asked for a bar narrow enough that an ordinary slug is not refused: secrets only. The delivered bar selects on the scanner hard-fail severity, and the scanner puts three kinds there rather than one, a secret pattern, a banned real name, and the caller's own local machine account name. The function comment says so plainly, so the width is disclosed rather than hidden, but it is wider than the ruling. The consequence is a refusal the author cannot act on: on a machine whose account name is an ordinary word, an ordinary page whose slug carries that word at a hyphen boundary is refused at ingest with a message telling the author to repair the slug at the source, when what matched was the machine account rather than anything in the page. That collision is already recorded against another surface as iss-2608291444328326, where the same rule turned the release payload gate red on a pristine tree until the home directory was pointed at an alias. It matters because a memory write is the one path here with no workaround: the page cannot be written under the name the distiller chose, and the remedy the message names is not the remedy. Fix direction: hold the filename to the secret patterns alone, as the ruling asked, or keep the identity kinds and make the refusal name which kind matched so an author can tell a collision from a leak. Detector: with the caller account name set to an ordinary word, an ordinary page whose slug contains that word must still write, while a page name carrying a token body is still refused.
