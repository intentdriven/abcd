---
id: spc-2609221017544877
slug: abcd-keeps-every-external-credential-the-same-way-one
intent: itd-2609221017023290
origin: researcher-authored
production_mode: hand-written
---
# abcd-keeps-every-external-credential-the-same-way-one

## Summary

The design record for itd-2609221017023290: the credential store, its three homes, the walkthrough and the single reader (adr-2609221017021499).

## Scope

1. **The package** `internal/core/credential`: `Resolve`, `Set`, the three home implementations (`external` resolves an environment variable or reads a named field of a named tool's configuration file; `abcd` reads and writes `~/.abcd/credentials.json` at 0600 through the atomic writer; `keychain` shells to the platform's keychain command on macOS and the secret-service tool on Linux, the configuration holding the name) (criteria 1, 3).
2. **The walkthrough** as a function itd-63's mode calls with the service's explanation and verification call; the CLI asks on the terminal, the plugin page through the host's question tool (criterion 2).
3. **The scanner** on the write path; a tracked-path write refused (criterion 3).
4. **The readers** switched to `Resolve`; a test greps adapters for direct environment or file reads of secret-shaped names (criterion 4).
5. **The record** names only names (criterion 5).

## Out of scope

- Rotation; sharing; the host's own credentials.

## Approach

The API adapter and the site setup land after this or in the same cut, reading through it from the first commit; the walkthrough is itd-63's mode with a credential step, not a second wizard.

## Footprint

- packages: internal/core/credential, internal/core/ahoy, internal/adapter/*
- tests: each home over fixtures; the refusal on an unset name; the tracked-path refusal; the reader grep; the record

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 unset refuses, no call | scope 1, 3 |
| 2 walkthrough with three homes | scope 2 |
| 3 no value in tree or harness; scanned | scope 3 |
| 4 one reader | scope 4 |
| 5 names only in the record | scope 5 |
