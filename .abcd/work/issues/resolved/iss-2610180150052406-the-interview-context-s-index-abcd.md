---
schema_version: 1
id: "iss-2610180150052406"
slug: "the-interview-context-s-index-abcd"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "build of the v0.15.0 deferred glossary findings (fix/deferred-glossary-flake)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/interview/README.md"
remedy: "Say the context holds the designed grill's terms (session, embark) and the knowledge floor every shipped interview is held to, naming the grill a design target."
resolution: "The interview context's README now says it holds the knowledge floor every interview is held to and the designed grill session's terms, and that the grill is a design target."
impact: internal
---

The interview context's index .abcd/development/brief/glossary/interview/README.md says the context holds terms specific to 'the grill/interview sub-verb of /abcd:intent', with session and embark among them (lines 3-5), presenting the grill sub-verb as shipped and conflating it with the interview sub-verb, though abcd intent --help lists no grill sub-verb and the session and embark entries call the grill a design target.
