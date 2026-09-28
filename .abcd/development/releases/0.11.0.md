# Release 0.11.0 (2026-09-26)

abcd's command list loses its modes-as-verbs and its five spellings of "check this repository": `abcd lint` is the check, with `lint docs`, `lint outbound`, `lint site` and `lint identity` as its targets, `ahoy` has the flags `--dry-run`, `--identity` and `--remote` instead of sub-verbs for its modes, and `--version` is where every tool keeps it. This is the release's break: the old spellings stop with exit 2 and name the new form, and `intent new` is gone. (itd-2609212130136102)

> "Twenty-four verbs, and three of them were the same check wearing different hats," said a product thinker reading `abcd --help`. "Now `lint` is the check, `ahoy` has flags instead of sub-verbs for its modes, and `--version` is where every tool keeps it. I can hold the list." (itd-2609212130136102)

`abcd --help` reads as a map instead of an alphabet: the person's verbs sit under five labelled headings (set-up, records, checks, portability, release), `--help --agent` adds the verbs agents and hosts call, and every verb's help opens with one sentence saying what it does, what it writes and when it refuses, identical on the list, the verb's `--help` and its page. The grouping and the sentences are gated, so a verb added without them fails a test. (itd-146, itd-2609212113220149)

> "I inherited the repo and typed `abcd --help` on day one," said Henry, a new hire. "I could read every line, so nothing was broken. What I could not do was tell which three of them I needed that morning, because alphabetical order puts `ahoy` next to `banlist` and tells you nothing about either." (itd-146)

> "The part I care about is that it cannot rot," said Kira, who maintains the surface. "If I add a verb and forget its group, the test tells me. If a group changes, the snapshot diff shows it. That is the same bar every other claim about this surface is already held to." (itd-146)

> "I read one line per verb before I decide whether to call it," said a technical facilitator watching an agent choose. "When that line says what the verb does, what it writes and when it refuses, the agent calls the right one. When it says 'manage things', it grep's the binary." (itd-2609212113220149)

Before a release is published, `abcd launch` renders the exact public payload through a default-deny filter that proves `.abcd/` never leaks, diffs it against the previously published release, smoke-tests every shipped command, skill and hook from the rendered snapshot, and runs the full pre-flight gate suite (identity layer, marker blocks, `plugin.json` and `marketplace.json`, dirty tree, documentation and hook compliance), so a publish is blocked on a finding, not merely previewed. (itd-65, itd-66)

> "The dry-run already tells me a home-directory path or a broken plugin.json *would* be a problem," said a maintainer. "But 'would' isn't 'does' — ship has to actually hard-fail on it. I don't want to hand-audit the payload before every snapshot; the gate suite should." (itd-65)

> "Before I publish the release I want to see the actual file list, be certain none of my development knowledge or flow state rode along, and know the plugin still works once it's just the shipped files," said a maintainer. "A preview I have to trust isn't enough — render it and prove it." (itd-66)

A repository abcd manages gets a release process that is correct the day it goes public: `abcd launch scaffold` lands a changelog-driven release gate armed against the reviewed content commit, and a `workflow_dispatch` rehearsal arms the full gate against a simulated release and publishes nothing, so a green rehearsal proves the gate works before it is trusted with a real tag. (itd-93)

> "I flipped my repo public and cut a release the same afternoon — it just worked," said Alice, a solo founder. "I didn't have to discover, the hard way, that my release gate could never be satisfied. abcd gave me the version abcd itself only reached after a day of untangling." (itd-93)

The conventions that are yours rather than abcd's now live once in `~/.abcd/rules.json` and inject into every repository abcd manages on that machine: the bundled opinions are the floor, your machine conventions refine them, and a repository override still wins. (itd-117)

> "I had the same three rules copy-pasted into several repos, and they'd already drifted — one spelled the trailer one way, another another," said Carol, maintaining a set of sibling projects. "The bundled defaults covered most of it, but the places where my house style differs from the tool's were exactly the places I was repeating myself. Now the delta lives in one file. When I change how we word a rule, I change it once, and the repo that genuinely needs to differ still overrides it locally." (itd-117)

Before abcd runs its own tests on your machine, it looks at what is already running: a program that has run flat out for over half an hour, or a machine loaded far beyond its cores, is named before the run starts (your own strays by name with how to stop them, other people's only as a count), and then the run carries on, because the choice to stop is yours. (itd-2609231434459890)

> "Two days of leftover busy loops took my machine down and nothing said a word," said Maya, an autonomous-development practitioner. "Now the first test run after I leave something burning tells me what it is and how to stop it." (itd-2609231434459890)

Type `abcd` in a terminal and it greets you with the a-b-c-d signal-flag hoist in colour, the version beside it and the tagline underneath, readable on light and dark terminals, in tmux and over SSH; pipe it into a file or a CI log and there is no banner and no escape byte. (itd-112)

Also in this release:

- The Registry Cannot Wave Its Hands (itd-122)
- A scope condition is dispositioned from a reading run, keyed to the condition's identity and joined to the item that occasioned it (itd-2609020625405251)

The line-by-line record of this release is its section in CHANGELOG.md.
