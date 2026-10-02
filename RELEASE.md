# Release 0.12.0 (2026-09-30)

Any role can run through a command-line harness the operator names, with the same brief, contract and transcript store as the host's own sub-agent: `roles.<role>.runner` routes a role to the claude CLI or opencode runner the machine enabled, and where the runner is absent or fails the host takes the role, a receipt names the role, the runner, the reason and the route that ran, and the run summary counts fallbacks per runner and per role. The runners are proven against stand-in harnesses so far; no real claude or opencode binary has yet run a role. (itd-2609201916056194)

> "My reviews were the scarcest thing in a run, and a second harness was sitting on the machine doing nothing," said a technical facilitator. (itd-2609201916056194)

An OpenAI-compatible API adapter reaches a configured provider for the roles and judgements pointed at it, and serves only the models that provider's list allows: the adapter takes a base URL and a key name, a model left off the list is refused before any call, abcd bundles no vendor denylist, and the run record shows the model that actually answered. With no provider configured nothing changes. (itd-2609081951381895)

> "I wanted one cheap decision model through OpenRouter, and I wanted to be certain nothing else of mine would ever go through it," said a product thinker configuring the first aggregator. (itd-2609081951381895)

One credential store, three homes, and every adapter reads through it: `abcd ahoy credential <name>` explains what a key unlocks and what works without it, then stores it where `--home` says, as a pointer to another tool's configuration or an environment variable, in the owner-only `~/.abcd/credentials.json`, or in the platform keychain, the home the explanation recommends. The abcd home refuses to write inside a git working tree. (itd-2609221017023290)

Bare `abcd` and the site's Status page show Now, Next and Later, computed from the record and maintained by nobody: Now is what is in a lane and the head of the pick order, Next is every planned intent the readiness gate reports ready, and Later is the rest, a count on the text board and rows in `--json` and on the site. (itd-2609212103568351)

> "The phase documents told me what someone once thought would happen next; this tells me what is next," said a product thinker reading the board after retiring phases. (itd-2609212103568351)

A change cannot call itself shipped while the brief lags the surface it delivered, and a release cannot be cut while an intent shipped since the last cut leaves its chapter behind: `abcd spec close` and the release cut refuse on a verb, sub-verb or agent no brief chapter names, and on a sentence a host-run reviewer confirms false, failing closed when no reviewer answers. `abcd docs fidelity` runs the same gate and can draft and apply the brief edit, flagged for review. (itd-60)

> "I read the verdict, not the source," said a product thinker shipping with abcd. (itd-60)

A cut release gets a retrospective: `abcd reflect <release-tag>` runs an interview seeded by the intents the tag shipped, their audit notes and the release's changelog section, and writes `.abcd/development/retrospectives/<release-tag>/README.md`, which a packed lifeboat carries with it. (itd-24)

> "abcd's brief and intents captured *what* I'd done," said Henry, a junior-developer persona. (itd-24)

Before the planning interview, a coherence pre-pass reads a draft against the brief's invariants, the principles and a one-line index of every other intent: `abcd intent prepass <itd-N>` writes the questions into the planning brief, each conflict quoting the invariant, each overlap with its four answers (keep both, bundle, supersede, refine), and any concern it cannot anchor asked as a question, never asserted as a conflict. (itd-42)

> "I'd grill an intent and it would come out crisp — clear terms, testable acceptance — and still be quietly redundant with something I'd specced two months earlier," said Iris, product lead. (itd-42)

A bot-opened dependency bump whose diff is only a manifest and its lock file can be re-authored as the repository's owner, so it merges on its own: `abcd launch scaffold --dependency-reauthor` writes the workflow, the message names the bot and the workflow with `Assisted-by: None`, the attribution gate is untouched, and everything else is left alone. The workflow is proven offline; its first live bump waits on the repository's GitHub App. (itd-2609221842494980)

> "Every bump sat blocked with auto-merge armed, looking as though it wanted a review no reviewer could give," said a product thinker who had just landed one by hand at the end of a long day. (itd-2609221842494980)

A live agent-session URL and a tool's generated-with footer are kept out of committed text: `abcd lint`, record-lint and `abcd lint outbound` refuse either shape, and the build loop strips both from the pull-request title and body it posts. Other posts are not scrubbed, so a pull request, issue or comment opened by hand is re-read and stripped after it is created. (itd-152)

Every place abcd writes names which of the two people it means, the product thinker or the technical facilitator, and the word maintainer leaves the vocabulary, refused by docs-lint. (itd-2609212137129937)

> "Agents kept asking 'the maintainer' and I never knew if they meant me deciding what to build or me running the gates," said a product thinker. (itd-2609212137129937)

Also in this release:

- An intent names the release it must land by, and the cut says whether it did (itd-2609212103572513)

The line-by-line record of this release is its section in CHANGELOG.md.
