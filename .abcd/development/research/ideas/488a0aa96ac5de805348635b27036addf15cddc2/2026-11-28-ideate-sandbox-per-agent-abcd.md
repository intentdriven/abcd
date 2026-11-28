# Ideate verdict — sandbox-per-agent-abcd

**Verdict: reframed.** Recorded on 2026-11-28 by abcd's idea-admission protocol —
primary-source research, a grill against the existing record, and an
independent adversarial review. This record exists so the idea is not
re-litigated: it stands whether the idea lived or died.

## The idea

Once the sandbox feature is ready, run each sub-agent in a dedicated sandbox with their own copy of the abcd CLI which offers all the functionality the sub-agent needs and/or has access to. In future, the abcd CLI binary can be configured to limit functionality for what the sub-agent is responsible to so to limit the danger of it going rogue.

## Where the idea came from

No source: the idea entered the gauntlet with nothing behind it but its own words.

## Leg 1 — Primary-source research

Every load-bearing claim checked against its primary source, never a
secondary citation. A claim marked quoted was checked only against the
source the idea came from, which proves the quotation and not the claim.

| Claim | Primary source | Finding |
|---|---|---|
| 1. The host harness (Claude Code) offers an OS-level sandbox for the commands an agent runs, with filesystem and network isolation, using Seatbelt on macOS and bubblewrap on Linux. — The doc says macOS 'uses Seatbelt', Linux and WSL2 use bubblewrap, and the sandbox has 'two independent layers': filesystem isolation and network isolation through a host proxy. It covers 'shell commands only' (Bash, PowerShell, Monitor and their children). Read, Edit, Write, WebFetch, MCP servers and hooks run outside it, and the doc says it 'is not a complete isolation boundary'. | https://code.claude.com/docs/en/sandboxing | verified |
| 2. That sandbox can be applied per sub-agent, each sub-agent in its own sandbox with its own filesystem view (versus session-wide only). — The Scope section says subagents 'run in the same process as the parent session and use the same sandbox configuration'. The sub-agents doc (https://code.claude.com/docs/en/sub-agents) has no sandbox frontmatter field; the nearest is 'isolation: worktree', a separate git worktree for the subagent's Bash commands, which is not an OS boundary. A separate per-agent sandbox exists only for a separate process: a background session is sandboxed by 'its settings', and the whole process can be wrapped in sandbox-runtime. | https://code.claude.com/docs/en/sandboxing | falsified |
| 3. Host sub-agent definitions can restrict which tools a sub-agent may use (the `tools:` frontmatter), and permission rules can restrict which Bash commands run. — `tools` is an allowlist and `disallowedTools` a denylist; to block specific commands while keeping Bash, the doc says to add a `permissions.deny` rule such as `Bash(git push *)`, which 'applies to the main conversation and to subagents'. The caveat is on https://code.claude.com/docs/en/permissions: a Bash rule 'isn't a security boundary around the program' (`/bin/rm`, `sh -c`, `git -C . push` slip past). The sub-agents doc also notes that a reviewer without Edit 'can still change files through Bash'. | https://code.claude.com/docs/en/sub-agents | verified |
| 4a. A capability-restricting CLI is not a security boundary when the agent also has a general shell; restriction holds only when the sandbox removes other routes. Least-privilege principle per Saltzer & Schroeder. — The paper defines least privilege ('Every program and every user of the system should operate using the least set of privileges necessary to complete the job') and complete mediation ('Every access to every object must be checked for authority'). A verb allow-list in abcd mediates only the calls that go through abcd, so a shell that reaches git and the Go toolchain gets around it. The page is a reproduction, revised April 1975 (Proc. IEEE 63(9), 1975). | https://www.cs.virginia.edu/~evans/cs551/saltzer/ | verified |
| 4b. OWASP LLM06:2025 Excessive Agency supports limiting functionality and enforcing authorization outside the LLM-controlled component. — It names three root causes: excessive functionality, permissions and autonomy. Its mitigations include minimising extensions and their functions, avoiding open-ended extensions, and 'Implement authorization in downstream systems rather than relying on an LLM'. Its example 'an extension to run one specific shell command fails to properly prevent other shell commands from being executed' is the case where the general shell undoes a restricted CLI. | https://genai.owasp.org/llmrisk/llm062025-excessive-agency/ | verified |
| 4c. Confused deputy (Hardy 1988) applies: a restricted abcd holding credentials could be steered to misuse its own authority. — Neither the ACM DL page (HTTP 403) nor Hardy's cap-lore.com page (certificate error) could be opened. Saltzer & Schroeder do not use the term, so it should not be attributed to them. | https://dl.acm.org/doi/10.1145/54289.871709 | unverifiable |
| 5a. Running agents in containers or microVMs with scoped credentials is current practice (Claude Agent SDK / Claude Code docs). — It covers sandbox-runtime, hardened Docker, gVisor and Firecracker microVMs. It recommends a least-privilege table and 'the proxy pattern' (a proxy outside the boundary injects credentials so 'the agent never sees the actual credentials'). https://code.claude.com/docs/en/sandbox-environments adds that the Bash sandbox alone 'is not sufficient for fully unattended runs'. | https://code.claude.com/docs/en/agent-sdk/secure-deployment | verified |
| 5b. Docker Sandboxes run coding agents in microVMs with credentials kept outside the VM. — It says 'Docker Sandboxes run AI agents in microVMs', each with a 'separate kernel per sandbox' and its own Docker Engine. Outbound TCP goes through a host proxy under a deny-by-default policy, and 'The raw credential values never enter the VM'. The page implies one sandbox per agent but does not state it. | https://docs.docker.com/ai/sandboxes/security/ | verified |
| 5c. OpenAI Codex sandboxes agent commands with platform-native OS mechanisms. — It uses Seatbelt on macOS and bubblewrap on Linux and WSL2, with read-only, workspace-write and danger-full-access modes. The page does not say whether Codex subagents get their own sandbox. developers.openai.com/codex/sandboxing redirects here (308). | https://learn.chatgpt.com/docs/sandboxing | verified |
| 6. A Go binary can enforce a compile-time or config-time verb allow-list. — Skipped as instructed: it is trivially true and not load-bearing, and no source was opened. What matters is claim 4: a build-tag or config allow-list holds only if the agent cannot reach the config file, the source tree, the Go toolchain or another abcd binary, which needs the sandbox's write and exec limits. | https://pkg.go.dev/cmd/go#hdr-Build_constraints | unverifiable |
| Added: per-agent OS boundaries in the host exist only at process granularity, so 'each sub-agent in its own sandbox' needs abcd to launch each agent as its own process (its runner route), not host in-process sub-agents. — The sandbox runtime 'wraps an entire process' and constrains 'the session's tools, hooks, and MCP servers'. The sandboxing doc says a background session 'runs in its own process' with its own settings. adr-2610161621415708 proposes (adr-2610161621415708 is a proposed ADR) this for agents abcd's runner launches. | https://code.claude.com/docs/en/sandbox-environments | verified |

## Leg 2 — Record grill

Does the brief, an intent, an ADR, or a principle already cover,
contradict, or supersede this idea? Every hit is cited by record id, and
every id resolved in this repository when the verdict was recorded.

| Record | Relation | Note |
|---|---|---|
| itd-2610161621364075 | covered | Planned intent (spec spc-2610161751056096 open, internal/core/fence not yet on origin/main) that is the 'sandbox feature' itself: every agent abcd's runner launches runs inside an OS-enforced fence confined to its own worktree and receipt folder, with abcd's guard teaching inside it. Covers the idea's first half for runner-launched agents; per-agent isolation is per launch, so each launched agent already gets its own fence. |
| adr-2610161621415708 | covered | Proposed ADR deciding that every agent the runner launches executes inside a whole-process OS boundary whose writable paths are derived per run, with abcd's guard kept inside as a mistake filter. It also records that drains and builds the person's own session drives (host-dispatched sub-agents) stay unconfined by abcd, which only reports their isolation, so 'each sub-agent in a dedicated sandbox' holds only for runner-launched agents. |
| dcn-2610161809259623 | covered | Picks Anthropic's sandbox-runtime as the fence for both the claude and opencode runners at the one launch point, wrapped per launch with a per-launch settings file behind the internal/core/fence seam, and admits abcd's own executable (the approver) into the fence as srt is admitted. This is the per-launch sandbox the idea waits for, with abcd's binary inside it. |
| spc-2610161751056096 | covered | The fence spec says the abcd inside each fenced launch is the starting abcd's own executable, pinned by content, an absolute path no launch may write, never looked up on PATH and never 'go run ./cmd/abcd' from the agent-editable worktree. That covers 'their own copy of the abcd CLI' inside the sandbox as a pinned, tamper-proof binary, though it is the full binary and not a per-role cut. |
| adr-2610222222326104 | covered | Accepted ADR: every agent a no-host drive starts goes through the runner's one launch point, so the sandbox fence applies to each one; its 2026-11-04 amendment makes the wall an allow-list of named tools with abcd's guard hook inside. Covers applying the sandbox to every sub-agent abcd itself starts. |
| itd-2611040344170206 | covered | Draft: each agent abcd starts unattended holds only the tools one allow-list names, re-read before each stage, which a repository may narrow and only the person may widen, and a stage run as a host sub-agent gets file and shell tools only. That is least privilege per agent for the harness's tools (not abcd's verbs); it states a profile is not a sandbox and defers real isolation to itd-2610161621364075. |
| dcn-2610161624025493 | covered | Each headless role's contract already decides which tools the launch grants: a role whose contract grants the shell has every shell prompt answered by abcd's guard, and a role granted no shell keeps dontAsk, refusing every prompt. Per-role limiting of what a launched agent can do exists at the tool layer, not the abcd-verb layer. |
| itd-2611011813155166 | covered | Draft for switching an abcd verb or project gate off by a declared key on the dormant pattern, read through the shared resolver, with no code removed from the binary. It is the configuration mechanism the idea's second half needs, but at repository scope set by the owner, not per sub-agent role; its decision 1 also gives protections that guard the person (shell guard hazards, secret scan, runner sandbox) no off switch. |
| adr-2610240850193281 | covered | Proposed ADR: a session's rights inside abcd are per-capability grants (push-branch, open-pr, decide-as) with presets, set only by the person, which a repository may narrow and never widen, and which are embedded in the runner's fence where one exists. That covers limiting what an agent may do through abcd by capability, and it states plainly that a grant is a permission, not a boundary. |
| itd-2610161710316538 | contradicted | Part B of the sandbox split: drains and builds the person's own session drives (including host-dispatched sub-agents) cannot be fenced by abcd, because abcd never edits the host's settings, so they get an isolation report instead of a sandbox. This contradicts 'each sub-agent in a dedicated sandbox' for sub-agents the host, not abcd's runner, starts. |
| adr-42 | contradicted | adr-42 (accepted): checks made at abcd's command layer are a mistake filter, not a security boundary, and anything needing a boundary needs an execution-layer control. A role-limited abcd binary is that kind of check, so the idea's claim that it limits 'the danger of it going rogue' cannot be made as a security property; only the OS sandbox can make it. |
| dcn-2610162040034117 | contradicted | Records that a harness started by a route outside the approver's list, such as another program name or a copied binary, is bounded by the OS sandbox, not by the list. By the same reasoning, an agent with a shell inside its fence can reach full abcd functionality by another route (a copied binary, a build from source), so verb-limiting the binary is a mistake filter and not a containment of a rogue agent. |

### Searches

| Terms | Places | Count |
|---|---|---|
| sandbox | .abcd/development/intents; .abcd/development/decisions/adrs; .abcd/development/principles; .abcd/development/research/notes; .abcd/development/research/ideas; .abcd/work/decisions; .abcd/development/brief; .abcd/work/issues/open | 66 |
| least privilege; least-privilege | same eight buckets | 3 |
| per-agent; per agent | same eight buckets | 49 |
| per-role; per role | same eight buckets | 25 |
| allowlist; allow-list; allow list | same eight buckets | 117 |
| verb allow | same eight buckets | 0 |
| restricted cli; restricted binary; restricted verb | same eight buckets | 0 |
| capability; capabilities | same eight buckets | 241 |
| allowedTools | same eight buckets | 4 |
| permission template | same eight buckets | 3 |
| fence | same eight buckets | 193 |
| command guard | same eight buckets | 3 |
| mistake filter | same eight buckets | 19 |
| profile | same eight buckets | 93 |
| switches a verb; switch an abcd verb | same eight buckets | 1 |
| gate off | same eight buckets | 2 |
| own copy of abcd/binary/cli | same eight buckets | 0 |
| copy of abcd; copy of the abcd | same eight buckets | 11 |
| limit the functionality; limit verbs | same eight buckets | 0 |
| rogue | same eight buckets | 1 |
| sub-agent; subagent | same eight buckets | 173 |
| role ... verb; role ... binary | same eight buckets | 16 |
| headless | same eight buckets | 58 |
| person-act | same eight buckets | 2 |
| person-only verb/act/command | same eight buckets | 6 |
| refuses ... from an agent | same eight buckets | 3 |
| role's contract; role contract | same eight buckets | 3 |
| tools a role / the role | same eight buckets | 3 |
| verbs ... an agent may/can/cannot | same eight buckets | 1 |
| which verbs | same eight buckets | 3 |
| administrative verb; privileged verb | same eight buckets | 0 |
| abcd binary/executable near fence/sandbox | same eight buckets | 0 |
| session-bin; abcd's own executable; admitted as srt; CheckPinnedBinaries | same eight buckets | 4 |
| session-bin; selfBinary; CheckPinnedBinaries | .abcd/development/specs | 3 |
| per-launch; per launch | same eight buckets | 17 |
| kill switch | same eight buckets | 18 |
| dormant | same eight buckets | 24 |
| enforcement-claims-are-facts | same eight buckets | 41 |
| restricted shell | same eight buckets | 2 |
| denylist; deny-list; deny list | same eight buckets | 39 |
| go run ./cmd/abcd | same eight buckets | 31 |
| internal/core/fence/ (code present) | origin/main tree | 0 |
| sandbox | ~/.abcd.noindex/ideas/488a0aa96ac5de805348635b27036addf15cddc2/ (private verdicts) | 13 |
| least privilege; own copy of; binary per; restricted cli/binary; limit verbs/functionality; role-scoped; verb allowlist | ~/.abcd.noindex/ideas/488a0aa96ac5de805348635b27036addf15cddc2/ (private verdicts) | 1 |

## Leg 3 — Adversarial review

Conducted fresh-context and off-policy by an evaluator that did not carry
out the research and received the idea as an artefact of unknown
authorship — the evaluator-outside-the-loop principle applied to ideas.
Each reviewer's attempts, verdict and sources are kept under its own label.

Current state: ignored. The reviewers judged the idea as an ambition for
the product this repository builds: a gap between the idea and what the
product ships today was out of scope as a kill, and reads as what the idea
implies building.

### Reviewer reviewer-a — reframed

- **fatal** — The second half's security claim. The idea says a cut-down abcd binary limits 'the danger of it going rogue'. A copy inside the sandbox has no authority beyond the sandbox's own. A fenced agent that has a shell reaches git, the network the fence allows, the files it may write, and in abcd's own repository the abcd source and the Go toolchain to build a full binary. So removing verbs from that copy takes away nothing the agent cannot reach another way. adr-42 rules that command-layer checks are a mistake filter and not a boundary. dcn-2610162040034117 applies the same reasoning to a copied binary, which only the OS sandbox bounds. Saltzer and Schroeder's complete mediation and OWASP LLM06's 'authorization in downstream systems' both put the control outside the component the agent controls. This is a contradiction in principle with an accepted ruling, not a gap in what ships today.
- **partial** — 'Each sub-agent in a dedicated sandbox' cannot cover the sub-agents the host dispatches. Host sub-agents run in the parent's process under the parent's sandbox configuration, and per-agent OS boundaries exist only per process. abcd also never edits the host's settings (adr-2610161621415708, itd-2610161710316538), so drains and builds driven from the person's own session get an isolation report, not a fence. The ambition still holds for every agent abcd's runner launches, which is the unattended case where the risk is highest (adr-2610222222326104 routes every no-host drive through that one launch point). It does not hold for 'each sub-agent' as written.
- **partial** — Novelty: is the first half already decided? Yes. itd-2610161621364075, adr-2610161621415708, dcn-2610161809259623 and spc-2610161751056096 already give each runner launch its own per-launch fence, with the starting abcd's own pinned executable inside it. The first half restates the planned sandbox intent and adds no new ambition. That does not make it wrong, but it means it is not a new idea.
- **survived** — Competitors converging. Claude Code's sandbox-runtime, Docker Sandboxes (one microVM per agent, credentials kept outside the VM) and Codex's Seatbelt and bubblewrap modes are making the per-agent sandbox a commodity. abcd survives this because it does not compete on the mechanism: adr-2610161621415708 puts the mechanism behind a replaceable fence seam (srt today, a container or microVM later). What abcd adds is the policy derived for each run (writable paths taken from the run's worktree and receipt folder), the load and machine checks made outside the fence, and signing outside the fence (dcn-2610222041439951). None of the competitors supplies that layer.
- **partial** — Fit with the rulings already on record. Per-role limits on what an agent may do already have four homes: per-capability grants set only by the person, which a repository may narrow and never widen, and which are stated to be a permission and not a boundary (adr-2610240850193281); verb and gate off-switches at repository scope (itd-2611011813155166); a per-stage allow-list of tools (itd-2611040344170206); and the tool grants in each role's contract (dcn-2610161624025493). A per-sub-agent verb allow-list in the binary would be a fifth permission system working alongside those. It would need to be folded into the grant model rather than added beside it.
- **survived** — Whether the need is real: does a fenced agent need abcd inside its fence at all? Yes. The approver that answers the guard's prompts has to run inside the fence (dcn-2610161809259623), and agents call record verbs and guard verbs. Putting a pinned abcd inside the sandbox is a real need, and limiting functionality according to the role remains a sound least-privilege goal (OWASP LLM06's first root cause is excessive functionality).
- **partial** — Wording. 'Its own copy of the abcd CLI' names the very route dcn-2610162040034117 says escapes the list, and it conflicts with the spec's rule that the abcd inside is the starting abcd's executable, pinned by its content at a path no launch may write, never a copy. 'Needs and/or has access to' blurs what a role needs with what it can reach, and least privilege depends on keeping those two apart. 'Going rogue' names an intent-driven threat, when the record's threat model is mistakes and steered (injected) behaviour, which a mistake filter does address. The idea must not describe the cut-down binary as containment (enforcement-claims-are-facts, adr-2610161621415708).

What the idea implies building:

- internal/core/fence and its srt adapter: one whole-process fence for each launch (spc-2610161751056096 is open and not on origin/main)
- A configuration for each role or sub-agent that switches abcd verbs off, read through the shared resolver
- A broker channel through which a fenced agent asks an abcd outside the fence, which holds the credentials, to push, open a pull request, reserve or land, with the request judged against the person's grant
- Grant presets for each role or stage, built on the per-capability grant in adr-2610240850193281
- The isolation report for drains and builds driven from the host (itd-2610161710316538)

Sources consulted: https://code.claude.com/docs/en/sandboxing, https://code.claude.com/docs/en/sub-agents, https://code.claude.com/docs/en/permissions, https://code.claude.com/docs/en/sandbox-environments, https://code.claude.com/docs/en/agent-sdk/secure-deployment, https://docs.docker.com/ai/sandboxes/security/, https://learn.chatgpt.com/docs/sandboxing, https://genai.owasp.org/llmrisk/llm062025-excessive-agency/, https://www.cs.virginia.edu/~evans/cs551/saltzer/, adr-42, adr-2610161621415708, adr-2610222222326104, adr-2610240850193281, itd-2610161621364075, itd-2610161710316538, itd-2611011813155166, itd-2611040344170206, dcn-2610161809259623, dcn-2610162040034117, dcn-2610161624025493, dcn-2610222041439951.

### Reviewer reviewer-b — reframed

- **fatal** — Nothing to take away. Inside the planned fence, the abcd binary runs with exactly the agent's OS rights and holds no extra authority. The launch carries no credential, the forge is off the host list, reservations and pushes are made by the loop outside the fence, and signing happens at land (spc-2610161751056096, dcn-2610161809259623). The verbs that could do real harm (reserve, land, announce, implement widen) already fail inside the fence because the OS layer withholds network and credentials. What is left are verbs that write local files, such as capture and decide, and the agent's shell can write those files directly. Cutting them from a role's binary removes no ability a rogue agent needs.
- **fatal** — Bypass by another route. Reads stay open inside the fence (srt's default; only credential stores are denied). So an agent can copy or run any other abcd it can read: the plugin-root binary, one on PATH, or one built with `go build ./cmd/abcd` from a checkout that contains abcd's source. It can also simply do with git and the shell what a cut verb would have done. This is exactly the case adr-42 and dcn-2610162040034117 decided: a check at abcd's command layer is a mistake filter, and only the execution layer bounds the agent. Saltzer and Schroeder's complete mediation fails because only calls that go through abcd are checked. OWASP LLM06's 'one specific shell command' example is the same failure.
- **partial** — What a prompt-injected agent can still do with a role-cut CLI: plant a backdoor in the worktree code, which is its deliverable and which abcd itself later signs and lands from outside the fence; write misleading receipts or briefs for the loop outside the fence to act on (the confused-deputy channel); read anything not on the deny list (e.g. ~/.claude.json, other repositories); and send data out through the model endpoint, which the spec admits stays reachable. A verb set inside the fence touches none of these. The real blast radius is set by what the privileged abcd outside the fence accepts from each role, not by which verbs the confined abcd offers.
- **partial** — False comfort. The idea's stated goal is to 'limit the danger of it going rogue'. A per-role binary named that way invites people to treat it as containment. The risk is worst where there is no fence at all: sub-agents dispatched by the host, which abcd cannot fence and only reports on (itd-2610161710316538, adr-2610161621415708). Without an OS boundary, the restricted CLI would be the only visible control, and it is bypassable. adr-2610240850193281 already insists that a grant is a permission, not a boundary. This idea would need the same disclaimer, which undercuts its own pitch.
- **partial** — Duplication. Per-role limits already exist at three layers. Role contracts decide which tools a launch gets (dcn-2610161624025493). The per-stage allow-list of named tools, which a repository may narrow and only the person may widen (itd-2611040344170206), does the same for each stage. Per-capability grants inside abcd, embedded in the fence (adr-2610240850193281), limit acts like push-branch and open-pr. A further per-role verb list would be a fourth place that encodes 'what a role may do', and the four could disagree.
- **partial** — Cost of keeping the lists in step. abcd has more than 40 top-level verbs plus sub-verbs, and they keep changing. adr-42 records that hand-kept enumerations rot, to the point where a size assertion certified a list that was wrong in both directions. Per-role verb sets are another such list. The attack is blunted because an allow-list fails closed: a new verb a role is not given is refused loudly and does not slip through, so drift shows up as broken lanes, not as holes. Paying that upkeep for a mistake filter is still poor value, unless the list is derived from the role contract rather than kept by hand.
- **survived** — Steelman: defence in depth, a smaller surface and clearer audit. These survive, but only in a weaker form. Within an OS fence, a role-scoped verb set does cut down the mistakes an honest-but-confused agent can make, such as a judge running capture or decide. It also makes 'agent called a verb outside its role' a sharp, cheap signal of injection or drift worth logging. That is real value for a mistake filter and for detection. It is not containment, and the log is incomplete because shell actions bypass it.
- **survived** — Steelman: 'its own copy of the abcd CLI in each sandbox'. This half is not killable, because it is already decided and stronger than the idea asks. The fence admits the starting abcd's own executable, pinned by content, at an absolute path no launch may write, and never found on PATH or run with `go run` (spc-2610161751056096, dcn-2610161809259623). That gives a tamper-proof approver and guard inside each per-launch fence.

What the idea implies building:

- The fence itself (internal/core/fence and its srt adapter) is not yet on origin/main. The idea's first half waits on itd-2610161621364075 and spc-2610161751056096.
- A per-role profile of abcd verbs: either a per-role build of the binary or a config-time verb allow-list read by the pinned binary. Neither exists. The nearest is itd-2611011813155166's switch that turns a verb off for a whole repository.
- A way to put host-dispatched sub-agents in sandboxes of their own. The host gives sub-agents the session's sandbox (they share its process), so this needs abcd's runner to launch each one as its own process.
- A guard rule, or a parent-side check, that refuses or logs a role calling an abcd verb outside the set derived from its role contract, by every route abcd can be invoked through.
- Validation on the outside of the fence of what each role's receipt and output may ask the outside abcd to do. The confused-deputy surface has to be closed there.

Sources consulted: https://code.claude.com/docs/en/sandboxing, https://code.claude.com/docs/en/sub-agents, https://code.claude.com/docs/en/permissions, https://code.claude.com/docs/en/sandbox-environments, https://code.claude.com/docs/en/agent-sdk/secure-deployment, https://www.cs.virginia.edu/~evans/cs551/saltzer/, https://genai.owasp.org/llmrisk/llm062025-excessive-agency/, https://docs.docker.com/ai/sandboxes/security/, adr-42, adr-2610161621415708, adr-2610222222326104, adr-2610240850193281, dcn-2610161809259623, dcn-2610161624025493, dcn-2610162040034117, itd-2610161621364075, itd-2610161710316538, itd-2611040344170206, itd-2611011813155166, spc-2610161751056096.

The reviewers agree, so no ruling was needed.

## Rejected alternatives

- **A per-role restricted abcd CLI as the control that stops a rogue agent** — Inside the sandbox the agent's shell can copy, rebuild or bypass abcd and reach git directly, and adr-42 with dcn-2610162040034117 rules a command-layer check a mistake filter, not a boundary; the OS sandbox and the outside abcd are the containment.
- **Separate per-role abcd binaries built with fewer verbs** — adr-21 builds abcd as a single statically linked Go binary, and a per-role build multiplies what each release must attest and keep in step; the proposed adr-2611011813195749 points the same way for code loaded at run time.
- **A dedicated sandbox for each sub-agent the host session dispatches** — The host runs its sub-agents in the session's own process under the session's one sandbox configuration (https://code.claude.com/docs/en/sandboxing), so a sandbox per sub-agent exists only for agents abcd launches as separate processes; itd-2610161710316538 plans an isolation report for the host-dispatched ones instead.

## The reframing

Each agent abcd's own runner launches runs in its own OS sandbox with abcd's pinned binary inside it, as itd-2610161621364075 already plans; containment comes from that sandbox and from the abcd outside it, which holds the credentials and checks what each role's output may ask of it. A per-role command set, derived from each role's existing contract and enforced by abcd's guard, survives only as a mistake filter that refuses and logs a call outside the role as a warning sign, never as a security boundary. Sub-agents the person's own host session dispatches share that session's sandbox and get the isolation report itd-2610161710316538 plans, not a sandbox of their own.

## What follows

The idea as posed does not survive, but the reframing recorded above
does. Any graduation to a draft intent carries the reframing, not the
original wording.
