---
schema_version: 1
id: "iss-2611040555039034"
slug: "06-delivery-01-build-sequence-md-55-56"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md"
remedy: "Say install asks at a terminal and takes flags or an answers file off one, and that some questions are terminal-only."
---

06-delivery/01-build-sequence.md:55-56 says abcd ahoy install carries its configuration as flags rather than a wizard. Install runs a question flow: 04-surfaces/01-ahoy.md:905-917 and commands/ahoy.md 'The setup flow' describe its plan screen and questions, --answers answers it from a file, and a --yes run ends partial, sending the commit checks, identity, routing table and drain rule to a terminal run; none of those has a flag.
