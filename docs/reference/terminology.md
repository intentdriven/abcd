# Terminology crosswalk

How established agentic-AI vocabulary maps onto abcd. Each row gives a term, a
one-line established meaning cited to a primary source, and abcd's position as
exactly one of four labels:

- **USES**: The concept is native; the row names the verb, surface, or
  principle that embodies it.
- **ADAPTS**: abcd holds the same ground under a different, sharper name; the
  row says what is sharpened and why.
- **REJECTS**: abcd deliberately does not do this, for a reason the
  development record states.
- **WATCHING**: The position is genuinely open; the row names the record entry
  tracking it.

Admission is deliberately strict: a term appears only with a primary anchor —
an official specification, a standards body, a peer-reviewed paper with a DOI,
or engineering documentation from the vendors where the term originated. No
single-author coinages, no aggregator write-ups. Identifiers such as `itd-N`,
`iss-N`, `adr-N`, and `spc-N` refer to entries in abcd's development record,
which travels with every repository checkout and release source archive but
never with the released binaries; native terms are glossed inline rather than
deep-linked.

## Index by theme

- **Protocols**: Agent Payments Protocol (AP2) · Agent2Agent Protocol (A2A) ·
  Model Context Protocol (MCP)
- **The core loop**: Agent loop · Autonomous agent · Multi-agent systems ·
  Orchestration · Tool use
- **Context**: Context engineering · Grounding · Memory ·
  Retrieval-augmented generation (RAG)
- **Safety**: Guardrails · Human-in-the-loop · Prompt injection · Sandboxing
- **Governance**: Agent identity · Authorization · Observability ·
  Policy-as-code · Tamper-evidence
- **Operations**: Agent harness · Agent skills · AGENTS.md ·
  Durable execution · Evaluations (evals)

## The crosswalk

| Term | Established meaning | abcd's position |
|------|---------------------|-----------------|
| **Agent harness** | The infrastructure around the model — the agent loop, tools, context management, and execution logic — that turns a model into a working agent.[^harness] | **USES** — "the agent harness" is abcd's own generic term for the host it configures and drives. abcd is a host-agnostic configuration layer over any harness, and a written adoption rubric (itd-51, draft) gates which autonomous harnesses abcd will drive. |
| **Agent identity** | Cryptographic workload identity issued across heterogeneous environments (SPIFFE); no ratified standard yet uses the exact term "agent identity".[^id] | **WATCHING** — iss-62 tracks guaranteeing the user's chosen git identity on every commit in managed repos; a permission/authorisation matrix is recorded as out of scope for agent coordination and routed to draft permission-template work (itd-33, itd-18). No workload-identity standard is adopted. |
| **Agent loop** | The iterative cycle in which a model interleaves reasoning with tool calls, feeding observations back until it produces an answer (the ReAct pattern).[^react] | **ADAPTS** — abcd's unit is the *autonomous run* (adr-27): iterate over ready work, gate each step on a receipt, apply a safety guard. The loop itself is supplied by a pluggable adapter; abcd defines the contract, never owns the loop. |
| **Agent Payments Protocol (AP2)** | Open protocol for agent-led payments: cryptographically signed intent, cart, and payment mandates carrying a user's authority to transact (v0.2; standardisation continuing at the FIDO Alliance).[^ap2] | **WATCHING** — the record is silent on payments, and that silence is captured as iss-138 rather than dressed up as a rejection. Adjacent commerce protocols are noted in the footnote. |
| **Agent skills** | A lightweight open format extending agent capabilities: a folder with a `SKILL.md` file plus optional scripts and assets; the specification is unversioned, draft-stage, and without a standards-body steward.[^skills] | **REJECTS** — the record rules that abcd ships zero skills: its surface namespace is commands only, and anything that mutates state is a command by definition (brief, internals: skills). The format stays under watch (iss-139) for the case a findings-only skill ever ships. |
| **Agent2Agent Protocol (A2A)** | Open standard for communication between independent, opaque agent systems: capability discovery, task management, secure exchange (v1.0.0, a Linux Foundation project).[^a2a] | **WATCHING** — a draft record (itd-33) leans against it: cross-agent coordination is "a small contract, not an orchestration substrate", with no agent-to-agent chat. Until that design is adopted, the position stays open. |
| **AGENTS.md** | A simple open format giving coding agents a predictable, project-specific place for guidance — "a README for agents" — stewarded by the Agentic AI Foundation under the Linux Foundation.[^agentsmd] | **USES** — an abcd-managed repo can carry an abcd-managed block in its AGENTS.md conventions entry point, maintained by the install verb; the block names abcd, so the install writes it only where the repo chooses that target, and by default into no conventions file. The format's foundation stewardship is tracked in iss-140. |
| **Authorization** | The process of verifying that a requested action or service is approved for a specific entity.[^authz] | **WATCHING** — deterministic gates decide what enters the record, but a permission model for agent actions is undesigned: the coordination draft rules an authorisation matrix out of its own scope and routes it to per-project permission templates (itd-33, itd-18 — both drafts; iss-62 covers identity). |
| **Autonomous agent** | A system situated in an environment, sensing and acting on it over time in pursuit of its own agenda (the classical taxonomy).[^auto] | **ADAPTS** — autonomy in abcd is bounded by construction: runs iterate under receipts and deterministic gates, and product decisions — adoption, sign-off, anything irreversible — stay human (*verifier selects, gates decide*: a verdict is a proposal; the human's adoption is the gate). |
| **Context engineering** | Building dynamic systems that curate the optimal set of tokens — information and tools, in the right format — supplied to a model at inference time (popularised mid-2025 in vendor engineering documentation).[^ce] | **USES** — the modular rules loader (shipped; itd-3) is exactly this: prompt-triggered recall matching injects only the rule domains relevant to the turn, and a prompt matching nothing injects zero tokens. |
| **Durable execution** | Making code fault-tolerant by automatically persisting execution progress, so a failed process resumes from the last completed step instead of restarting.[^durable] | **WATCHING** — planned run resilience (itd-29): checkpoint on failure, resume from the native spec store — deliberately file-based and framework-free, and not yet shipped. |
| **Evaluations (evals)** | Systematic tests of model outputs against specified criteria; the peer-reviewed anchor is holistic, multi-metric evaluation.[^evals] | **USES** — a standing discipline: no judge ships unmeasured (itd-81) — calibration corpora with at least 40% known-good cases, scoring on recall *and* true-negative rate, and a false-positive ceiling that gates the prompt lock. The same discipline records a rejection of judge panels as redundant votes. |
| **Grounding** | Supplying a model use-case-specific information at inference time so responses connect to accurate, verifiable sources.[^ground] | **ADAPTS** — abcd's version is *cite-or-be-dropped* (adr-35): a claim in a synthesised artefact carries its source citation or does not ship, and whatever could not be grounded is reported loudly as coverage blanks rather than papered over. |
| **Guardrails** | Programmable controls on an application's inputs and outputs — validation, filtering, dialogue constraints — enforced outside the model.[^guard] | **ADAPTS** — abcd's equivalents are deterministic gates: lint families, receipt gates, and pre-commit guards all fail closed. The pre-tool-use shell-hazard guard (`abcd guard`) deliberately does not — it is *fail-open-loud*, a mistake filter rather than a security boundary (adr-42): a guard that cannot answer warns unmissably and lets the command run, because a broken guard must never brick a session. Two principles sharpen the industry sense: *enforcement claims are facts* (a claimed gate must exist) and *guards prove themselves* (a guard ships with the test that shows it fires). |
| **Human-in-the-loop** | Human-oversight capability for intervention in an AI system's decision cycle; EU law's term of art is "human oversight".[^hitl] | **ADAPTS** — *verifier selects, gates decide*: a model verdict is only ever a proposal; admission is decided by deterministic gates plus the human's adoption. Planning an intent is defined as the human's sign-off act, machine-checked before implementation may start. |
| **Memory** | An agent's capability to retain and retrieve information beyond one context window: short-term session context plus long-term knowledge persisted across sessions.[^mem] | **USES** — the per-project memory substrate (`.abcd/memory/`) with its own verb: raw sources are curated into a compounding knowledge artefact by a single writing curator, and retrieval is recall-matched and budget-bracketed rather than context-flooding. |
| **Model Context Protocol (MCP)** | Open JSON-RPC protocol standardising how LLM applications connect to tools and data through hosts, clients, and servers (specification revision 2025-11-25).[^mcp] | **USES** — MCP is one of the four recorded oracle-adapter shapes, and an MCP front door over the transport-agnostic core is a recorded later step (adr-24, adr-25). No MCP server ships today; the architecture is built so that adding one changes no core code. |
| **Multi-agent systems** | Multiple interacting autonomous agents coordinating to solve tasks beyond any individual agent's capability — three decades of distributed-AI literature that current usage narrows.[^mas] | **ADAPTS** — abcd's multi-agent shape exists for *independence*, not parallelism: the evaluator-outside-the-loop principle requires that whoever judges a change is not whoever proposed it. Cross-agent work coordination is a draft design (itd-33). |
| **Observability** | A common telemetry schema for GenAI systems — spans, metrics, and events for inference, agents, and tools; the semantic conventions are pre-stable (pinned here at v1.41.1, the last tagged release carrying them).[^otel] | **WATCHING** — the record has a native, local, redacted transcript corpus (adr-29) and advisory run telemetry in a planned intent (itd-29); model-effectiveness tracking is a draft (itd-17). No tracing or metrics stack is adopted. |
| **Orchestration** | Coordinating which agents run, in what order, and how control and results flow between them.[^orch] | **ADAPTS** — abcd is *host-delegated* (adr-25): the deterministic core prepares work and hands a prompt to the host's own dispatch. abcd owns the prompt; the host owns models, credentials, and execution — the orchestration-substrate role is deliberately declined. |
| **Policy-as-code** | Expressing enforcement policy in declarative, machine-evaluated form, decoupling policy decisions from application logic; the policy *engine* is the component that decides grant or deny.[^pac] | **USES** — abcd's policy is committed JSON configuration: per-repo rules, docs-lint, record-lint, and backend-seam selection, evaluated deterministically in the core. The record practises this without using the phrase; this page is where the mapping is made. |
| **Prompt injection** | A vulnerability where user prompts or ingested content alter an LLM's behaviour in unintended ways — directly or indirectly (OWASP LLM01:2025).[^pi] | **USES** — recorded defences on both sides of the boundary: agents that read untrusted input must carry injection-canary fixtures (the itd-5 discipline), and a verb that mutates state fails closed on anything not exactly recognised. The canary lint (reserved code PQ006) and automated canary execution are recorded design targets, not yet shipped. |
| **Retrieval-augmented generation (RAG)** | Combining a generator with a retriever over an external index, so generation draws on non-parametric knowledge.[^rag] | **ADAPTS** — the sources corpus takes the script-first shape: per-source folders, extracted text, grep-based consult. It is a user-tier script MVP, not shipped product behaviour (absorption into the core is tracked as iss-27); retrieval is recorded as a pluggable seam (iss-26) where a RAG backend is one opt-in adapter, never the default. |
| **Sandboxing** | An OS-enforced boundary restricting an agent's filesystem and network access, so it can act autonomously without unrestricted host access.[^sand] | **ADAPTS** — abcd's containment is structural rather than OS-level: read-only probes proven by before-and-after tree hashes, a destination safety gate that refuses directories abcd did not produce, and refusals that write nothing. OS-level sandboxes belong to the host. |
| **Tamper-evidence** | Append-only log integrity via Merkle trees, so any instance of a log can be proven a superset of any earlier instance.[^tamper] | **WATCHING** — receipts are hash-anchored and manifests verified today; a compliance-grade hash chain over conversation and edit history is a draft (itd-16), and tamper-evident receipts are tracked as iss-141. |
| **Tool use** | A model emits structured, schema-conformant calls to declared functions; the application executes them and returns results ("function calling" in some vendors' vocabulary).[^tooluse] | **ADAPTS** — abcd sits on the other side of the mechanism: it *is* the tool. A single binary of verbs over a transport-agnostic core that returns structured results and knows nothing about who called it (adr-23); thin front doors render those results per surface. |

## Sources

[^harness]: Chen, C., "Unlocking the Codex harness: how we built the App Server", OpenAI Engineering, 4 February 2026, https://openai.com/index/unlocking-the-codex-harness/; Lopopolo, R., "Harness engineering: leveraging Codex in an agent-first world", OpenAI Engineering, 11 February 2026, https://openai.com/index/harness-engineering/; Anthropic, "Building agents with the Claude Agent SDK", 29 September 2025, https://claude.com/blog/building-agents-with-the-claude-agent-sdk; Anthropic, "Effective harnesses for long-running agents", 26 November 2025, https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents. <!-- docs-lint: allow -->

[^id]: SPIFFE Project (CNCF), "SPIFFE Standard", stability: Stable, https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE.md. An IETF individual draft composing SPIFFE, WIMSE, and OAuth 2.0 for AI agents exists (draft-klrc-aiagent-auth) but carries no formal standing.

[^react]: Yao, S., Zhao, J., Yu, D., Du, N., Shafran, I., Narasimhan, K., and Cao, Y., "ReAct: Synergizing Reasoning and Acting in Language Models", ICLR 2023, DOI 10.48550/arXiv.2210.03629. The loop shape is defined independently in OpenAI's Agents SDK ("Running agents", https://openai.github.io/openai-agents-python/running_agents/) and Anthropic's tool-use documentation (https://platform.claude.com/docs/en/agents-and-tools/tool-use/how-tool-use-works).

[^ap2]: "Agent Payments Protocol Specification", v0.2, https://ap2-protocol.org/ap2/specification/; launch announcement, Google Cloud, 16 September 2025, https://cloud.google.com/blog/products/ai-machine-learning/announcing-agents-to-payments-ap2-protocol; contributed to the FIDO Alliance, 28 April 2026, https://fidoalliance.org/fido-alliance-to-develop-standards-for-trusted-ai-agent-interactions/. Adjacent: x402 (under Linux Foundation governance, https://x402.org/) and the Agentic Commerce Protocol (OpenAI and Stripe, https://www.agenticcommerce.dev/) — the latter's "ACP" abbreviation collides with IBM's Agent Communication Protocol, which merged into A2A in August 2025 (https://lfaidata.foundation/communityblog/2025/08/29/acp-joins-forces-with-a2a-under-the-linux-foundations-lf-ai-data/).

[^skills]: "Agent Skills — Specification", https://agentskills.io/specification; specification repository, https://github.com/agentskills/agentskills (Apache-2.0 code, CC-BY-4.0 documentation). Unversioned, with no formal draft/final designation; at least one field is marked Experimental; no standards-body stewardship is stated on the primary sources.

[^a2a]: "Agent2Agent (A2A) Protocol Specification", v1.0.0, https://a2a-protocol.org/latest/specification/; Linux Foundation project launch, 23 June 2025, https://www.linuxfoundation.org/press/linux-foundation-launches-the-agent2agent-protocol-project-to-enable-secure-intelligent-communication-between-ai-agents.

[^agentsmd]: AGENTS.md, https://agents.md/ (repository: https://github.com/agentsmd/agents.md); a founding project of the Agentic AI Foundation, Linux Foundation, 9 December 2025, https://www.linuxfoundation.org/press/linux-foundation-announces-the-formation-of-the-agentic-ai-foundation.

[^authz]: NIST, CSRC Glossary, "authorization" (the process sense, per SP 800-152), https://csrc.nist.gov/glossary/term/authorization.

[^auto]: Franklin, S. and Graesser, A., "Is It an Agent, or Just a Program?: A Taxonomy for Autonomous Agents", ATAL 1996 (Springer LNCS 1193, 1997), DOI 10.1007/BFb0013570. See also Wooldridge and Jennings under multi-agent systems.

[^ce]: LangChain, "The rise of 'context engineering'", 23 June 2025, https://www.langchain.com/blog/the-rise-of-context-engineering; Anthropic, "Effective context engineering for AI agents", 29 September 2025, https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents.

[^durable]: Microsoft, "What is Durable Task?", https://learn.microsoft.com/en-us/azure/durable-task/common/what-is-durable-task; Temporal, "Understanding Temporal", https://docs.temporal.io/evaluate/understanding-temporal; Restate, "Key Concepts: Durable Execution", https://docs.restate.dev/foundations/key-concepts#durable-execution.

[^evals]: OpenAI, "Working with evals", https://developers.openai.com/api/docs/guides/evals; Anthropic, "Define success criteria and build evaluations", https://platform.claude.com/docs/en/test-and-evaluate/develop-tests; Liang, P., et al., "Holistic Evaluation of Language Models", Transactions on Machine Learning Research, 2023, DOI 10.48550/arXiv.2211.09110.

[^ground]: Microsoft, "Grounding Data Design for AI Workloads on Azure", https://learn.microsoft.com/en-us/azure/well-architected/ai/grounding-data-design; Google, "Grounding with Google Search", Gemini API documentation, https://ai.google.dev/gemini-api/docs/google-search. Distinct from the symbol-grounding problem of cognitive science. <!-- docs-lint: allow -->

[^guard]: Rebedea, T., et al., "NeMo Guardrails: A Toolkit for Controllable and Safe LLM Applications with Programmable Rails", EMNLP 2023 System Demonstrations, DOI 10.18653/v1/2023.emnlp-demo.40; OpenAI Agents SDK, "Guardrails", https://openai.github.io/openai-agents-python/guardrails/; AWS, "Amazon Bedrock Guardrails", https://docs.aws.amazon.com/bedrock/latest/userguide/guardrails.html.

[^hitl]: European Commission High-Level Expert Group on AI, "Ethics Guidelines for Trustworthy AI", 2019, DOI 10.2759/346720, https://digital-strategy.ec.europa.eu/en/library/ethics-guidelines-trustworthy-ai; Regulation (EU) 2024/1689 (the AI Act), Article 14 "Human oversight", https://eur-lex.europa.eu/eli/reg/2024/1689/oj/eng. ISO/IEC 22989:2022 is sometimes cited for this term, but its defined-terms clause does not include it.

[^mem]: AWS, "Add memory to your Amazon Bedrock AgentCore agent", https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/memory.html; Google, "Memory: Long-term knowledge with MemoryService", Agent Development Kit documentation, https://adk.dev/sessions/memory/; research antecedent: Park, J. S., et al., "Generative Agents: Interactive Simulacra of Human Behavior", UIST 2023, DOI 10.1145/3586183.3606763. <!-- docs-lint: allow — US spelling inside a quoted paper title -->

[^mcp]: Model Context Protocol, "Specification", revision 2025-11-25, https://modelcontextprotocol.io/specification/2025-11-25; stewarded by the Agentic AI Foundation, a directed fund of the Linux Foundation, since 9 December 2025, https://www.linuxfoundation.org/press/linux-foundation-announces-the-formation-of-the-agentic-ai-foundation (donation announcement: https://www.anthropic.com/news/donating-the-model-context-protocol-and-establishing-of-the-agentic-ai-foundation).

[^mas]: Wooldridge, M. and Jennings, N. R., "Intelligent agents: theory and practice", The Knowledge Engineering Review 10(2), 1995, DOI 10.1017/S0269888900008122; Dorri, A., Kanhere, S. S., and Jurdak, R., "Multi-Agent Systems: A Survey", IEEE Access 6, 2018, DOI 10.1109/ACCESS.2018.2831228.

[^otel]: OpenTelemetry, GenAI semantic conventions, pinned at semantic-conventions v1.41.1 (May 2026), https://github.com/open-telemetry/semantic-conventions/releases; the dedicated successor repository, https://github.com/open-telemetry/semantic-conventions-genai, declares `Status: Development` and has no tagged release as of July 2026.

[^orch]: OpenAI Agents SDK, "Agent orchestration", https://openai.github.io/openai-agents-python/multi_agent/; Microsoft, "Semantic Kernel Agent Orchestration", https://learn.microsoft.com/en-us/semantic-kernel/frameworks/agent/agent-orchestration/; Anthropic, "Building effective agents", https://www.anthropic.com/engineering/building-effective-agents.

[^pac]: Open Policy Agent documentation (a CNCF graduated project), https://www.openpolicyagent.org/docs; NIST, SP 800-207, "Zero Trust Architecture", 2020, DOI 10.6028/NIST.SP.800-207 (defines the policy engine as the component that decides access).

[^pi]: OWASP Gen AI Security Project, "LLM01:2025 Prompt Injection", OWASP Top 10 for LLM Applications 2025, https://genai.owasp.org/llmrisk/llm01-prompt-injection/.

[^rag]: Lewis, P., et al., "Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks", NeurIPS 2020, DOI 10.48550/arXiv.2005.11401.

[^sand]: OpenAI, Codex documentation, "Sandboxing", https://learn.chatgpt.com/docs/sandboxing; Anthropic, "Configure the sandboxed Bash tool", https://code.claude.com/docs/en/sandboxing. Both sources note that sandboxing needs filesystem *and* network isolation, and reduces rather than eliminates risk. <!-- docs-lint: allow -->

[^tamper]: IETF, RFC 9162, "Certificate Transparency Version 2.0", December 2021, DOI 10.17487/RFC9162, https://www.rfc-editor.org/info/rfc9162/ — cited as the transferable verifiable-log primitive, not as an agentic-AI standard.

[^tooluse]: OpenAI, "Function calling", https://developers.openai.com/api/docs/guides/function-calling; Anthropic, "Tool use with Claude", https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview; Google, "Function calling", Gemini API documentation, https://ai.google.dev/gemini-api/docs/function-calling. <!-- docs-lint: allow -->
