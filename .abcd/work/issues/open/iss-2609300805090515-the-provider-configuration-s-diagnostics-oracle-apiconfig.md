---
schema_version: 1
id: "iss-2609300805090515"
slug: "the-provider-configuration-s-diagnostics-oracle-apiconfig"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/oracle/connect.go"
remedy: "Carry the configuration read's diagnostics out of oracle.Connect on its result (a field the JSON omits, or a named diagnostics member) and print each on stderr in the ahoy connect front door, the way ahoy credential does; give detectProviderAdapter a non-required gap naming each skipped route. Grounds: the APIConfig doc comment already says the diagnostics are for a front door to print on stderr, and ruling CD2 asks for the skip to be said; a test per door with a repository route to a keyed provider shows it."
---

The provider configuration's diagnostics (oracle.APIConfig.Diagnostics: a role outside the roster, a route to an unconfigured provider, and a repository's route to a keyed provider skipped under ruling CD2) reach the person only through 'abcd ahoy --providers' (its board) and 'abcd ahoy credential' bare (stderr). 'abcd ahoy connect' (oracle.Connect loads the configuration and drops them) and the bare 'abcd ahoy' provider-adapter gap (detectProviderAdapter) say nothing, so a skipped route is silent there.
