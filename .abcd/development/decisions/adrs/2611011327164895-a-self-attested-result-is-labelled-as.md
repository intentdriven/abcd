---
id: adr-2611011327164895
slug: a-self-attested-result-is-labelled-as
status: proposed
date: 2026-11-01
supersedes: null
superseded_by: null
related_intents: [itd-2611011325428560, itd-2610261025516272]
related_rfcs: []
related_adrs: []
---

# ADR-2611011327164895: A self-attested result is labelled as such and only the person can switch it on

## Context

A private repository pays hosted CI by the minute, and abcd's agents land work often, so re-running in hosted CI the gates the person's machine has just run is the largest avoidable cost a solo maintainer meets (itd-2611011325403127 measures it). The cheap alternative, posting the local result to the forge, is exactly what the forge cannot check: any person or integration with write access can set any commit status (GitHub's protected-branch documentation), artifact attestations are produced only inside GitHub Actions (and need Enterprise Cloud for a private repository), and SLSA v1.2 places a build run on the developer's own machine at Build L0, its provenance at L1 "trivial to bypass or forge". The agents run as the person's own account: the interactive guard does not protect the person's machine configuration from an agent's writes, and the agents' forge login holds admin rights (iss-2610170725144804). One ruling stands and one is proposed: no receipt minted in the cloud is trusted (adr-2610120937435856, accepted), and the person alone grants autonomy (adr-2610240850193281, proposed).

## Decision

We will let a repository's local gates be reported to the forge only as a self-attestation, labelled as such, and only when the person has switched it on by an act abcd's own verbs refuse to its agents; it stands in for a hosted run only once it is posted by an identity the agents cannot use, which is open until the planning interview rules on it. The posted result carries a name of its own that says it is self-attested (never the name an independent check reports under) and a description saying it ran on the person's machine and was not independently verified. The switch is set with the person present, by a touch of a security key or a fingerprint on a key the agents' processes cannot use without it, and is recorded on the person's machine; no verb abcd offers an agent turns it on; but agents run as the person's own account, so an agent with write access to abcd's own code or to the recorded grant could defeat the switch itself, and the switch therefore bounds abcd's own landing, not such an agent. It is offered only where the conditions hold: a private repository, one maintainer, a person as technical facilitator, and changes by the person or their agents; a change from anyone else always gets the full independent check. Wherever abcd reports a landing, a self-attested one is shown as such.

## Alternatives Considered

- **Labelled self-attestation behind a switch only the person can set (chosen).** Honest because the label says what the result is, and safe enough because the switch needs the person present. Its residual, stated plainly: posted from the person's token, the status can be posted by any agent holding that token without touching the switch at all, so the switch bounds abcd's own landing, not the forge; the label stays truthful about what kind of result it is, but it asserts that the local gates passed, which a forged post makes false. Until an identity the agents cannot use posts it (open question 1 of itd-2611011325428560), it is a label beside the hosted run, never a required check that replaces it.
- **A plain commit status from the person's token (gh-signoff's pattern).** The cheapest, but the agents hold the same token, so the status is the agent's word, and it would reopen the required-check pinning of iss-2610290913450522. Rejected.
- **Cryptographic attestation the forge verifies.** GitHub verifies no locally produced test evidence; artifact attestations need Actions. Not available, so not an option today.
- **No self-attestation at all; always run hosted CI.** Honest and simple, and stays the default; rejected as the only choice because its cost makes abcd unaffordable for a solo private repository.

## Consequences

- A solo maintainer of a private repository can land with no hosted run, knowingly, once the self-attested result is posted by an identity the agents cannot use; until then it is a label beside the hosted run.
- abcd must guard the switch as far as an account it shares with its agents allows: the grant needs the person's touch, and since an agent with write access to the grant record or to abcd's code could still defeat it, the record says so wherever it shows a self-attested landing.
- Every landing report distinguishes self-attested landings from independently checked ones, so the product thinker can see what no independent check examined.
- The independent checker of itd-2611011325430933 is the upgrade path: the same person can move from self-attestation to a check no agent can fake without changing how they land.
