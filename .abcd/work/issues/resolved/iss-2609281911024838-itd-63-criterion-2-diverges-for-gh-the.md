---
schema_version: 1
id: "iss-2609281911024838"
slug: "itd-63-criterion-2-diverges-for-gh-the"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-63"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/detect.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker: should abcd offer to install GitHub's command-line tool itself, on an explicit yes, when ahoy remote or site setup needs it, or does itd-63 say the install half covers gitleaks only? The first is a lane: both verbs route their missing-gh refusal through tools.Install with the host-relayed yes."
remedy: "Waits on the itd-63 gh ruling: if offered: ahoy remote and site setup route a missing gh through tools.Install with the host-relayed yes (gh offered only when a verb that needs it runs, by the registry's Homebrew step) and installToolNames accepts gh, proven by tests that nothing installs without the yes and that the yes runs the install; if narrowed: record the narrowing in the H10 shape, never rewriting criterion 2: this issue resolved by the lane's commit plus an Audit Notes line on itd-63 saying the install half covers gitleaks only."
resolution: "Resolved on the DQ3 ruling (offer to install gh on an explicit yes): ahoy remote apply and site setup explain a missing gh and offer the registry's Homebrew step through ahoy.OfferGH. The step runs only on a yes typed at a terminal. --yes, a piped answer and a run with no terminal decline, and the refusal carries the command. A failed or unverified install refuses before any request. ahoy --remote never installs and names the apply instead. --install-tool still names gitleaks alone. itd-63 gains the criterion-2 Audit Notes line under ruling H10."
impact: additive
resolved_by:
  commit: "e9a393675"
---

itd-63 criterion 2 diverges for gh: the explain-then-install mode's install half is offered for gitleaks alone. ahoy.DependencyTools is {gitleaks} (internal/core/ahoy/detect.go), so ahoy install never puts gh to the question and --install-tool gh is refused by installToolNames (internal/surface/cli/cli.go), while ahoy remote and site setup refuse a missing gh with tools.Missing (internal/core/ahoy/remote.go) and show the Homebrew step the person must then run by hand. A required tool the registry knows how to install is explained but never installed on a yes, so for gh the criterion's 'the install runs only on an explicit yes' never arises. Either a gh dependency gap joins DependencyTools (offered only when a verb that needs gh is in play), or the record states that the mode installs gitleaks only.

## Remedy grounds (2026-09-29)

SOTA check: the gh project names Homebrew as its recommended macOS route (https://github.com/cli/cli/blob/trunk/docs/install_macos.md, read 2026-09-29), which the registry already explains; mise refuses to act on a project's config until the person trusts it (https://mise.jdx.dev/cli/trust.html, read 2026-09-29), the same consent-before-install shape as the host-relayed yes. Rejected for fit: adopting mise or devbox as the installer, a new dependency the repository has not signed off.

## Grounds

- pursued: a person at a terminal who meets a missing gh at ahoy remote apply or site setup is asked, and on yes gh is installed and the verb goes on. TestRemoteApplyOffersGhAndRunsTheStepOnYes and TestSetupOffersAMissingGh show it, and TestAhoyRemoteApplyNeverInstallsGhOnAScriptedYes shows a scripted yes installs nothing. It is shown wrong if a --yes or piped run ever reaches the install step, or a yes at a terminal still leaves the verb refusing with gh uninstalled.
