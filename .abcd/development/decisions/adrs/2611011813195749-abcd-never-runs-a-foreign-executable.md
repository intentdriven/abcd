---
id: adr-2611011813195749
slug: abcd-never-runs-a-foreign-executable
status: proposed
date: 2026-11-01
supersedes: null
superseded_by: null
related_intents: [itd-62, itd-113]
related_rfcs: []
related_adrs: [adr-21, adr-22, adr-2610241057124151]
---

# ADR-2611011813195749: abcd never runs a foreign executable under its own name, and no code that writes records is loaded at run time

## Context

People asked for abcd to be extended by turning each verb into a plug-in or module.

The verbs are not independent features. They are views over one record store, and that store has shared rules: the lint gates, the release cut and the reservation fence read what capture, intent and spec write. So a verb and the gate that judges its output have to read every field the same way.

None of the ways to load code at run time keeps that guarantee or abcd's trust chain intact:

- **Go's `plugin` package.** The plug-in must be built with exactly the same toolchain, and exactly the same source for every shared dependency. A loaded plug-in cannot be closed, and the race detector supports it poorly (https://pkg.go.dev/plugin).
- **Executables found by name on PATH** (`git-<name>`, `gh-<name>`). Their hosts verify no provenance; GitHub says outright that extensions are not verified, signed or endorsed (https://cli.github.com/manual/gh_extension).
- **Subprocess RPC** (HashiCorp's go-plugin). Its handshake "is not a security measure", and checksum checks are opt-in (https://github.com/hashicorp/go-plugin).
- **WebAssembly sandboxes.** By default they deny the file and process access that a verb needs.

SLSA provenance covers only the artefacts its build produced (https://slsa.dev/spec/v1.0/threats). abcd updates itself only from builds whose provenance is signed by its own release workflow (iss-2609231050273096), and it promotes its cache to PATH only through a home-scoped attestation (adr-2609151706587280), and it fetches tools only from downloads pinned in its own release (adr-2610241057124151).

## Decision

We will keep the verbs compiled into the one attested binary (adr-21). No code that writes records is loaded at run time, and abcd never runs a foreign executable under its own name.

abcd is extended at capability seams instead (adr-22):

- A repository declares, in data, which external program stands behind a seam, pinned by checksum and signer.
- The person admits that program once from their home before it runs on their machine. The admission is bound to the repository's identity (its root commit), the program's checksum and signer, and the exact invocation and configuration the repository declares for it, so a yes given for one repository is never reused by another, and a changed argument or config asks again. Admitting is a person-act: the shell guard refuses it from an agent, and no sequence may contain it.
- The core's joins see the adapter's result unchanged: build, drain, lint, launch, the guard and the board.
- The first seam is the project check of itd-62.

A verb-shaped addition for a repository's product lives in the host's own plugin format or in a script, and reaches abcd through the MCP front door's structured results (itd-113).

## Alternatives Considered

- **Each verb a Go `plugin`.** Rejected because of the exact-build rule: every plug-in would need rebuilding for each abcd release. That suits modules built alongside abcd, and those are already Go packages.
- **`abcd-<name>` executables on PATH, as git and gh do it.** Rejected. Nothing verifies them, they sit outside every join the core makes, and they would duplicate the host's own plugin mechanism under abcd's name.
- **Subprocess RPC or WebAssembly verbs.** Rejected. The first adds a dependency and a process on every hook event. The second denies what verbs need. Both still need a trust chain per plug-in.
- **Capability seams declared in pinned data. Chosen.** The joins stay in the core, and only the backend varies.

## Consequences

- An extension never changes what a record means, so every repository's ledger stays readable by `embark`, `lint` and the release cut.
- Each new seam is built only when a consumer needs it. The project check of itd-62 is first.
- The person's one-time admission needs a home-scoped store keyed on the repository's root commit, the program's checksum and signer, and its declared invocation and configuration, as the existing one-time allows are keyed on the root commit. It is asked again when any of them changes.
- A wish to add a verb becomes a contribution to the one release, a host plugin, or a script. abcd itself never provides a plug-in channel.
