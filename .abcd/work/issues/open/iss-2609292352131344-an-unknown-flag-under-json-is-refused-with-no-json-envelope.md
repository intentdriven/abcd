---
schema_version: 1
id: "iss-2609292352131344"
slug: "an-unknown-flag-under-json-is-refused-with-no-json-envelope"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
remedy: "In Run(), when the error is a flag-parse usage error and the persistent json flag reads false, fall back to the raw args: a literal --json (or --json=true) before any -- terminator asks for the envelope; test abcd ahoy --bogus --json yields one envelope on stdout, exit 2, nothing on stderr. Grounds: the envelope contract abcd's own --json help states and iss-29 / iss-2609100519128005."
---

An unknown flag under --json is refused with no JSON envelope on stdout: abcd ahoy --bogus --json (and any verb) prints cobra's 'abcd: unknown flag: --bogus' on stderr, exits 2 and leaves stdout empty, because the flag parse fails before the persistent --json flag is read, so Run() sees json=false. The --json contract ('a refusal is a {"abcd":"error",...} object on stdout too') is broken for every flag-parse refusal; a machine caller passing an old flag (abcd version --check --json after v0.12.0) reads nothing.
