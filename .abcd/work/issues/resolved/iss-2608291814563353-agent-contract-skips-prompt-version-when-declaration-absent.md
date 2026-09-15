---
schema_version: 1
id: "iss-2608291814563353"
slug: "agent-contract-skips-prompt-version-when-declaration-absent"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/lint/agentcontract.go"
resolution: "checkAgentTrustContract records the missing-declaration finding and falls through to the prompt_version check; a test lints a prompt whose frontmatter carries only name: and asserts both findings in one run"
impact: fix
---

ultra-v0.6.8 B2: checkAgentTrustContract in internal/core/lint/agentcontract.go returns early when reads_untrusted_input is absent, so the prompt_version check below it never runs — contradicting its own comment that prompt_version is required for EVERY prompt. checkAgentChangelog then skips the page as already reported, which is untrue on this path: a prompt with only name: in its frontmatter yields one finding instead of the declaration finding, the missing prompt_version finding and the changelog finding. Fix: record the declaration finding and fall through.
