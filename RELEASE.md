# Release 0.11.1 (2026-09-28)

abcd makes sure the right person is the author of record before a single commit lands. `abcd ahoy` checks the identity a commit would actually carry, author and committer, whatever git config or environment produced it, against the identity the repository pins; when they diverge it proposes the pinned (or global) identity and asks before writing repo-local config, and with no one to ask it reports and writes nothing. (itd-131)

> "A sandbox `Test User` override authored 54 commits before anyone noticed — we had to rewrite history and force-push to unpick it," said Alice, who maintains the repo. "Now `ahoy` catches a wrong identity before the first commit, and it asks me rather than guessing." (itd-131)

A repository abcd manages declares what it ships, and the release flow follows the declaration. A Go application or a macOS app with its own tag-driven workflow declares its artefact kind once, `launch --dry-run` and `launch ship` run against it with the changelog-driven gate, the deferral read and the derived version, and `launch scaffold` lays the gate beside the release workflow the repository already has and leaves that workflow as it was. (itd-2609150819432059)

One verb sets up a managed repository's site: `abcd site setup` writes the composition, the render-then-deploy workflow and the environments and prints the step left, and with a hosting credential configured it creates and routes the host and reports the address. The site is abcd's own page set, rendered from that repository's record. (itd-2609061543533170)

> "The explorer, the graph, the timeline: I had them for abcd and wanted them for every repository abcd manages," said a product thinker looking at the record browser. "Now one verb writes the workflow and the environments, and when I have given abcd a hosting credential it creates and routes the host too. Any managed repo's site looks like abcd's, with its own record in it." (itd-2609061543533170)

A new issue, or a draft intent filed as quoted text, is matched against the record at filing: a likely double is linked and named, and never dropped. (itd-2609212137116617)

> "I filed the same finding twice a month apart and nobody noticed until a consistency pass," said a technical facilitator. "Now the capture tells me at filing that it looks like iss-N, writes the link, and leaves it to me to confirm. Nothing is refused: a wrong match is a link I remove, not a finding I lost." (itd-2609212137116617)

The status-line badge always reads one of three states, `abcd-managed`, `waiting on the product thinker` or `waiting on the technical facilitator`; an agent's question to the human is refused until the mode says who is being asked, and the next human answer resets it. (itd-2609212130146198)

> "The badge was set by whichever agent remembered," said a product thinker who had watched it read managed while an agent waited on them. "Now an agent cannot ask me anything until it has said which of us it is asking, and the moment I answer the badge goes back. It is the one signal I have that something is waiting; now it is true." (itd-2609212130146198)

Consult any source freely and cite only by deliberate human choice: `abcd source` keeps a local-only corpus of material you are not free to name and an append-only ledger of what influenced which decision, citation needs the source's permission and your own flip of the ledger line, and `abcd source cite-check` clears text before it is shared, reporting offenders by key alone. (itd-76)

> "I could never let an agent near my working papers before, because one helpful footnote could burn a collaborator's trust," said Alice, a researcher-developer. "Now it reads everything, records what influenced what, and cites nothing. When the paper behind a decision is finally published, I flip one flag — and the whole influence trail is already written." (itd-76)

Every shipped intent owes a fidelity review, and abcd now says which ones are still owed: bare `abcd intent audit` lists each with its receipt and the command that re-emits the request, `abcd intent` carries the count, and `abcd intent audit --owed` hands a host the owed requests oldest first, leaving them owed when no reviewer is reachable. Nothing refuses on the debt. (itd-2609150819445595, itd-53)

> "I asked what was outstanding and got a list of eight, with the command to re-emit each one," said Iris, a technical facilitator paying down a run's review debt. "Last week the same question was a grep through the decision log." (itd-2609150819445595)

Intents that ship as one piece of work plan as a bundle: one shared spec, both records moving to `planned/` together and shipping together when the spec closes, with `abcd intent reclassify` to change a record's kind or supersede it. One glossary page now maps the record families (intent, spec, step, bundle, issue, release, status) and how each moves, with phase, milestone and roadmap marked superseded. (itd-34, itd-2609211913453478)

> "I kept asking which word to use, and every answer named a different document," said a product thinker who had just approved bundles and wondered whether phases still meant anything. "Now there is one page: intent, spec, step, bundle, issue, release, status. Phase and milestone are on it too, marked superseded, with what replaced them. I read it in five minutes." (itd-2609211913453478)

A review names the commit it read, and the bare `abcd` board shows how far the default branch has moved since, flagging one past twenty commits. `abcd intent consistency` checks the brief and the intents against each other and files each contradiction as an issue, quoting both ends. (itd-28, itd-48)

When a capability could use a program you have not installed, `abcd ahoy install` explains it first: what the program is, whether this capability needs it, what already works without it and the exact install step. It installs only on an explicit yes, and a no leaves the capability on its native default and says so. (itd-63)

Also in this release:

- An admission and a surprise are written by a verb, and the order the design fixes is a refusal (itd-2609020625400194)
- A reframe occasioned by a reading is recorded as a reframe, joined to what occasioned it, without carrying the construal it replaced (itd-2609020625402518)
- The scribe's context is assembled and its output is ingested by a verb, and the record can show that no session held both a reading and the ledger (itd-2609020625402599)
- A principle carries typed claims, its reference, its comparison and its evidence, its statement is readable cold, and it inherits only what held (itd-2609020625405170)
- abcd lab mechanises the lab conventions three hand-run experiments proved (itd-2609212137128014)

The line-by-line record of this release is its section in CHANGELOG.md.
