---
id: spc-2609221011153746
slug: abcd-ships-an-openai-compatible-api-oracle-adapter-the-first
intent: itd-2609081951381895
origin: researcher-authored
production_mode: hand-written
---
# abcd-ships-an-openai-compatible-api-oracle-adapter-the-first

## Summary

The design record for itd-2609081951381895: the OpenAI-compatible API adapter with per-provider allowlists and the vendor denylist (adr-2609221009491186).

## Scope

1. **Configuration** (`internal/core/oracle/config.go`): the provider blocks, the bundled denylist, the resolver that validates every role and judgement route at read time and returns the refusal (criteria 1 to 3).
2. **The adapter** (`internal/adapter/openaiapi`): one client over the chat completions protocol, the request rendered from the same brief the host gets, the response validated against the same output contract, the transcript captured (criterion 1).
3. **The credential**: by name from the machine configuration or the environment; a name that resolves to nothing is a refusal (criterion 4).
4. **The record**: provider, requested and reported model per call in the run record (criterion 5).
5. **The setup** (`ahoy` gap → itd-63's mode): detect no provider block; explain; on yes ask the home (three choices, keychain recommended in prose); write the key by the chosen method (`external`: store the variable or config name only; `abcd`: `~/.abcd/credentials.json` mode 0600; `keychain`: the platform keychain under abcd's service name via the `security` command on macOS and the secret-service API on Linux, the configuration holding the name); write the provider block with the first allowlist; one verification call (criteria 7, 8).
6. **Review**: security reviewer on the lane (criterion 9).

## Out of scope

- Provider routing; learned routers; the tier's judgement.

## Approach

The adapter implements the same validator/runner interface the host path and the RepoPrompt route (itd-6) implement, so the loop does not know which ran; OpenRouter and a local server differ only in configuration.

## Footprint

- packages: internal/core/oracle, internal/surface/cli, and a new adapter package for the OpenAI-compatible client
- tests: the resolver's refusals (unlisted, denylisted, no key); the call against a fake server; the record fields

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 the call; unconfigured unchanged | scope 2 |
| 2 unlisted refused at read | scope 1 |
| 3 denylist wins | scope 1 |
| 4 key by name | scope 3 |
| 5 the record | scope 4 |
| 7 ahoy explains and offers | scope 5 |
| 8 three homes, keychain in prose, nothing in the harness | scope 5 |
| 9 security review | scope 6 |
