---
schema_version: 1
id: "iss-2610150749053362"
slug: "several-draft-intents-carry-h1-titles"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "iss-2608220150157511 fix review pair"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/03-out-of-scope.md"
remedy: "Give each of those drafts a one-line H1 that names its user moment, keeping the paragraph as the opening of its press release, then regenerate the index; consider a record-lint warning on an H1 longer than a line."
---

Several draft intents carry H1 titles that read badly now that the out-of-scope drafts index is generated from each record's H1: itd-70's H1 is the placeholder 'Drawn-out intent', itd-2610070544145422's H1 is cut off mid-sentence ('…and it just works: the'), and eleven drafts have paragraph-length H1s of 300 to 1,093 characters (for example itd-2609150819440345 and itd-149).
