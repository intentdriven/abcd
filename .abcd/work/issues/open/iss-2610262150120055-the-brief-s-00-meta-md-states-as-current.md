---
schema_version: 1
id: "iss-2610262150120055"
slug: "the-brief-s-00-meta-md-states-as-current"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "ideate leg 3 on simplifying the brief: the adversarial review read embark_types.go"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/00-meta.md"
remedy: "Rewrite the 00-meta.md embark bullet to say what embark does today (brief/** is report-only), naming the reverse walk as unbuilt and pointing at the intent that would build it if one exists, or capturing that gap."
---

The brief's 00-meta.md states as current fact that /abcd:embark reads a lifeboat and writes a new repo's brief by walking the brief-lifeboat mapping in reverse, with the amended press release becoming the new repo's initial brief. Shipped embark does no such thing: internal/core/lifeboat/embark_types.go classes every brief/** file of a lifeboat as report-only (IgnoredReportOnly, reportOnlyPrefixes), never written into a target. The reverse walk is a design target, so the sentence is a staged claim written in the present tense, and it misled a session into telling the product thinker that every managed repository's brief is written from the lifeboat's shape.
