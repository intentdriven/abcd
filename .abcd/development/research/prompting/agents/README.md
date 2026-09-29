# Per-Agent Prompting SOTA Research

> Per the brief (§ "Research-driven prompts"): per-agent SOTA research is **task #1 of each agent's epic**. The output lives here as `<agent-name>.md` and references the baseline at [`../01-general-best-practices.md`](../01-general-best-practices.md).
>
> **Role.** Same as the baseline: research is the **gate** (audit reference for `lifeboat-reviewer` and the prompt linter), not the **source** (prompt template). The author writes the agent's prompt informed by the research; the oracle audits alignment.

## Purpose of these files

For each agent it covers, `<agent-name>.md` answers four questions specific to that agent's job:

1. **What is the closest prior art?** Existing prompts in public repos (Piebald, VoltAgent, EliFuzz, Cursor / Devin extracts), academic papers on the agent's task type, and any internal predecessors (e.g. the manual lifeboat for `flow-essence`).
2. **What are the agent-specific failure modes?** General failures from `01-general-best-practices.md` plus things that bite *this* agent specifically (e.g. context rot for `chat-distiller`, verbosity bias for `press-release-composer`, injection for `embark-scaffolder`).
3. **What techniques from the SOTA taxonomy fit this agent?** Named techniques from The Prompt Report (CoT, self-consistency, ReAct, plan-and-solve, etc.) with a *one-line* justification per technique.
4. **What golden-test fixtures should this agent have?** 2–5 input/output pairs that span the behaviour envelope, plus injection canaries where the threat model demands.

## File shape

Use `_template.md` as the starting point. Every per-agent research file MUST include:

- Frontmatter: `name`, `description` (one-line scope), `agent: <agent-name>`, `baseline: 01-general-best-practices.md`
- The four sections above (closest prior art / failure modes / SOTA techniques / fixture sketches)
- A short "Open questions" tail capturing things the author could not resolve from desk research and which need spike work or human judgement during the agent's epic

## When to update

- **Initial creation**: task #1 of each agent's flow-next epic.
- **Material drift**: when the baseline doc (`01-…`) supersedes to `02-…`, the periodic SOTA-audit oracle (B+C+D infrastructure, D component) flags any per-agent file whose pillars (failure modes, techniques, fixtures) are now inconsistent with the new baseline.
- **Post-launch incidents**: if a production failure traces back to a missing pitfall here, append a "Lessons learned" section.

Per-agent files are **immutable once their epic ships**. Subsequent additions go in a new section dated at the bottom of the file or, if structural, supersede with a new file (`<agent-name>-02.md`). Mirrors how the brief itself archives.

## Inventory

The files here are measured against two rosters: the agent prompts that ship
under `agents/`, and the design roster the brief records as still to be built
([`05-internals/01-agents.md`](../../../brief/05-internals/01-agents.md), "The
design roster still to be built"). A research file for a design target is
research done ahead of its agent, not a baseline for a prompt that ships.

| Research file | Agent | Standing against `agents/` |
|---|---|---|
| [`chat-distiller.md`](chat-distiller.md) | `chat-distiller` (Pass B) | Design target: no `agents/chat-distiller.md` ships, and the brief lists it on the roster still to be built. |
| [`embark-scaffolder.md`](embark-scaffolder.md) | `embark-scaffolder` (embark) | Design target: no `agents/embark-scaffolder.md` ships, and the brief lists it on the roster still to be built. |
| [`intent-fidelity-reviewer.md`](intent-fidelity-reviewer.md) | `intent-auditor` | Ships as [`agents/intent-auditor.md`](../../../../../agents/intent-auditor.md), renamed from `intent-fidelity-reviewer` by itd-123 (spc-28); the research file keeps the name it was written under. |

Every other shipped agent under `agents/` has no per-agent research file here,
and neither does any other design target.
