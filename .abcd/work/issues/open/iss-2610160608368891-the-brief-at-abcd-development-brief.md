---
schema_version: 1
id: "iss-2610160608368891"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
remedy: "Add --outbound, --readme-parts, --answers/--answered-in and --install-tool to the list of non-interactive flags at 01-ahoy.md:956."
---

The brief at .abcd/development/brief/04-surfaces/01-ahoy.md:956 says 'The non-interactive flags pre-answer the prompts:' and lists approve-everything, adopt, marker target, oracle backend, deep scan, visibility, dev mode, the stale-binary override, the PATH directory and the attribution hook as the whole set. `abcd ahoy install --help` also registers --outbound, --readme-parts, --answers, --answered-in and --install-tool, each of which pre-answers a prompt.
