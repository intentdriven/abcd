---
schema_version: 1
id: "iss-2609261909106167"
slug: "the-private-banlist-guard-in-githooks"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainS3"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-commit"
resolution: "The private-banlist pre-commit guard, and the scaffolded copy ahoy installs, read every staged line that holds a JSON string escape or a percent-encoded byte in its decoded spellings too (the JSON escape layers and the percent view), and refuse by key on a match in either; a decoder failure refuses and names the step."
impact: fix
resolved_by:
  commit: "e86097c46"
---

The private-banlist guard in .githooks/pre-commit reads every staged blob raw under LC_ALL=C, so a private name written with JSON string escapes passes it: a \uXXXX spelling of any letter, non-ASCII or plain ASCII, or a \/ inside a pattern, puts bytes in the blob that no ERE written for the plain spelling matches, and a JSON transcript, export or fixture is the natural carrier. The store-before-commit redactors and the lint rules read the scanner's decoded views (lineViews: the percent view and the JSON escape layers); the hook, the one enforcement point of the private layer, reads only the text as written. Detector: a keyed banlist entry for a fake name refuses a staged JSON file that spells the name with \u escapes or percent-encoding, exactly as it refuses the plain spelling.

## Grounds

- pursued: a keyed entry for a fake name refuses a staged file spelling it with unicode escapes, a nested escape layer, an escaped solidus or percent-encoding, exactly as it refuses the plain spelling, while escapes that spell no banned name pass; a staged escaped spelling of a banned name that commits cleanly would show it wrong
