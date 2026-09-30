---
schema_version: 1
id: "iss-2609300025372991"
slug: "two-shipped-intents-implementing-specs-sections-misstate"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
found_at: ".abcd/development/intents/shipped"
origin: researcher-authored
production_mode: hand-written
remedy: "Rewrite both sections to name the live spec_id and to say the native store reuses each number for another spec, qualifying every predecessor id '(predecessor store)' per the specs charter's Two spc-N Namespaces rule; grounds: the frontmatter spec_id and the files under specs/closed/ are the primary record, and itd-4's own spc-6 catch-up section is the shape to follow."
---

Two shipped intents' Implementing specs sections misstate where their predecessor-store spec ids stand: itd-36 says its frontmatter spec_id records spc-38 as the primary delivering spec, while the frontmatter records spc-2609211905174684 and live spc-38 and spc-39 are itd-136's record explorer and itd-137's relationship chart; itd-4 says its spc-20 to spc-23 do not exist in the native spec store, while the live store holds all four as other specs (banlist, fresh install, stale-binary warning, tier placement). Found while qualifying predecessor-store citations for iss-2609290448510918.
