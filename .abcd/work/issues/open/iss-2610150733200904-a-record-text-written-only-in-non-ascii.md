---
schema_version: 1
id: "iss-2610150733200904"
slug: "a-record-text-written-only-in-non-ascii"
severity: "nitpick"
category: "ux"
source: "review-followup"
found_during: "itd-2609151138388536 build review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/recordid"
remedy: "Word the refusal as 'nothing in the text can form a file name (ASCII letters or digits)' across the families that share the slug cutter, or transliterate before cutting; pick after a state-of-the-art check of slug transliteration."
---

A record text written only in non-ASCII letters, such as a decision in Cyrillic, is refused as having no letters or digits, which is false for that text; the slug cutter keeps ASCII only, and the message describes the slug's emptiness as the text's.
