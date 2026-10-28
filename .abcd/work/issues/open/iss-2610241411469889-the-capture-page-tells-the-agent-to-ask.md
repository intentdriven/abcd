---
schema_version: 1
id: "iss-2610241411469889"
slug: "the-capture-page-tells-the-agent-to-ask"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "autonomous run 2026-10-24, capturing the dependency-pin-verification issue"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/capture.md"
remedy: "The capture page's remedy step states the question layout or points at the question-drafter agent, and the loader recalls GRILL on a capture prompt that will need a question."
---

The capture page tells the agent to ask the product thinker for a missing remedy, but the prompt that ran it did not recall the GRILL domain, so the agent asked without the question layout and the question guard refused it on five parts before the remedy question could be shown
