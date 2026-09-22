# Ideate verdict — abcd-operator-console

**Verdict: reframed.** Recorded on 2026-09-22 by abcd's idea-admission protocol —
primary-source research, a grill against the existing record, and an
independent adversarial review. This record exists so the idea is not
re-litigated: it stands whether the idea lived or died.

## The idea

A control programme and monitor for abcd: a GUI application (desktop or local web) through which a product thinker configures abcd (providers, credentials, the model tier per role, the pace, which judgements run where) and monitors it (the Now / Next / Later status, lanes in flight, the run record, the badge state, owed reviews, the ledger), for one or many abcd-managed repositories, with abcd itself running as the binary alone and calling whatever harnesses are available (claude -p, opencode, others) — the ultimate goal being that a person can manage their repositories outside any harness, from this programme, while still working inside their preferred harness when they choose. The working name 'abcd CP/M' is a joke on Digital Research's mark and cannot be used; the idea needs a name of its own.

## Leg 1 — Primary-source research

Every load-bearing claim checked against its primary source, never a
secondary citation.

| Claim | Primary source | Finding |
|---|---|---|
| Claude Code can be driven headlessly by another program with structured output and non-interactive permissions | https://code.claude.com/docs/en/headless | verified |
| Without --bare, a claude -p run executes the project's hooks and MCP servers with no trust dialog | https://code.claude.com/docs/en/headless | verified |
| opencode can be driven headlessly, and offers a server with HTTP, SSE and an explicit permission endpoint | https://opencode.ai/docs/server/ | verified |
| OpenAI Codex can be driven headlessly, and its app-server speaks JSON-RPC with approvals as server-initiated requests | https://learn.chatgpt.com/docs/app-server | verified |
| At least one further harness (Gemini CLI, Cursor CLI, Aider) offers a headless mode a Go program could call | https://geminicli.com/docs/cli/headless/ | verified |
| A local-web GUI over the Go binary needs no new dependency (net/http plus embed), while every desktop shell costs cgo or a second toolchain | https://pkg.go.dev/embed | verified |
| Existing agent control planes manage sessions and diffs; none configures a policy layer above the harness (model tier per role, pace, where judgements run) | https://github.com/andyrewlee/awesome-agent-orchestrators | verified |
| A GUI control plane improves outcomes over a CLI plus a status page | https://arxiv.org/abs/2607.01418 | unverifiable |
| The CP/M trademark registration is live | https://tsdr.uspto.gov/statusview/sn73149955 | falsified |

## Leg 2 — Record grill

Does the brief, an intent, an ADR, or a principle already cover,
contradict, or supersede this idea? Every hit is cited by record id, and
every id resolved in this repository when the verdict was recorded.

| Record | Relation | Note |
|---|---|---|
| itd-2609201916151817 | contradicted | Decision 5 records the process driver as an opt-in reversal of the host-delegated boundary, confirmed for the opt-in path only; the idea made harness-less operation the destination. The loop is also non-resident (every invocation does one step and exits), so nothing is in flight between steps for a monitor to show. |
| itd-2609201916056194 | covered | The command-line runner the programme would drive; a draft whose unreachable path falls back to the host, which the harness-less goal removes. |
| itd-22 | covered | Any harness through an adaptor over one seam; the programme is a consumer of that, not a second copy. |
| itd-2609170822093401 | contradicted | Names the operator, not the product thinker, as the one who accepts the model tier, and keeps model choice and credentials with the host on the harness leg. |
| itd-167 | covered | Records the product thinker's decision of 2026-08-29 that their first stop surface is a web page they open when they have time, which supports a served pane rather than contradicting it. |
| itd-2609212103568351 | covered | The computed Now / Next / Later status the programme would render, already planned for the board and the site. |
| itd-2609061543533170 | covered | The rendered site already presents the record read-only for a managed repository. |
| itd-2609221017023290 | covered | The credential store with three homes is the configuration half; the programme edits through it rather than holding secrets of its own. |
| itd-113 | covered | The MCP front door is the third thin door and is still a draft; a fourth door is not prescribed by the architecture, only permitted. |
| itd-2609212130146198 | covered | The badge the programme would show is a status-line surface already planned. |

## Leg 3 — Adversarial review

Conducted fresh-context and off-policy by an evaluator that did not carry
out the research and received the idea as an artefact of unknown
authorship — the evaluator-outside-the-loop principle applied to ideas.

- **fatal** — The named user is wrong for most of the surface: an accepted decision record bars addressing the product thinker in the facilitator's register, and the ledger, run record and owed reviews are facilitator artefacts, as is the model tier's acceptance
- **fatal** — Making harness-less operation the ultimate goal inverts the host-delegated boundary whose reversal was confirmed only as an opt-in, and removes the host fallback two planned records depend on
- **partial** — No resident process exists to show lanes in flight; a monitor needs a first long-running verb or a second driver beside the binary's
- **partial** — One or many repositories presumes an enumeration abcd cannot do: the machine stores are keyed by root commit with no map back to a checkout
- **partial** — Script-first is skipped on top of three unshipped contracts (the runner, the routing table, the credential store)
- **partial** — A desktop shell breaks the single-binary and no-new-dependency rules; local web over the standard library does not
- **partial** — The credential pane can configure only the provider leg, since the host owns its own login, and a localhost UI that accepts secrets is reachable by any local process without a per-launch token
- **partial** — The idea states no falsifier, and no evidence exists that a GUI control plane beats a CLI plus a status page
- **survived** — The record rejects a web surface for the product thinker
- **partial** — Fanning claude -p across many repositories runs each repository's hooks and MCP servers with no trust dialog
- **partial** — A fourth front door is added before the third (MCP) exists

## Rejected alternatives

- **A desktop application (Wails, Fyne, Tauri or Electron)** — Every shell costs cgo or a second toolchain against the single-binary boundary, for no measured outcome gain; local web over the standard library costs nothing.
- **A product-thinker control programme that configures providers, credentials, tiers and judgement placement** — An accepted decision record bars addressing the product thinker in the facilitator's register; those items are the operator's, and the product thinker's pane carries stops, the plain-language verdict and the pace.
- **Harness-less operation as the destination, with no host in the picture** — The host-delegated boundary's reversal was confirmed as an opt-in only, and the runner's fallback lands on the host; ruled instead that when abcd runs as the binary with no host session, the fallback is a configured host of the person's choosing, and every fallback is recorded.
- **Many repositories from the start** — The machine stores are keyed by the repository's root commit with no map back to a checkout; multi-repository waits for a registered-checkout store under the user-level home.
- **Building the console before the runner and routing intents ship** — Script-first: the console's contract is exactly what those seams are still settling, and the rendered site plus the JSON the CLI emits are the script rung that discovers it.
- **Keeping the working name CP/M** — The trademark registration is dead, so the objection is not legal, but the name reads as a 1970s operating system to the audience and names nothing about what the thing does.

## What follows

The idea as posed does not survive, but the reframing recorded above
does. Any graduation to a draft intent carries the reframing, not the
original wording.
