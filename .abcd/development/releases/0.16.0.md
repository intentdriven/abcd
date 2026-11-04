# Release 0.16.0 (2026-10-31)

**What you must change: an ideate verdict proves every leg it records.** `abcd ideate record` now takes only a version-2 verdict, and a version-1 payload is refused with a message naming version 2. Every record id in every leg resolves, a primary source must be a web address, a filed source or a record, the grill says what it searched and where so an empty grill can be told from a broken search, a reframing has its own field, two reviewers are recorded reviewer by reviewer, and a claim checked only against the idea's own source reads "quoted", never "verified". A host or script that writes verdicts moves to the shape the `ideate` command page shows; verdicts already on record stay as they are. (itd-2610141646454347)

A session told to work on while its person is away keeps working until they are back. The person says once that sessions should work on, until 09:00 say, and abcd records it for the repository and repeats it to every session at each start, after each compaction and with every prompt. A session that tries to end its turn while it still has work it can do alone is handed the next item; a question it would have put to the person is decided and recorded; an act only the person may approve is queued for their return. The mode itself grants nothing. (itd-2610180912189125)

A repository says who merges, and abcd keeps to it. A repository declares whether its technical facilitator is a person or abcd. With a person, a maintainer's ready pull request waits for the facilitator's approval and abcd arms the merge only once it is given, whoever asks; with abcd, a maintainer's pull request that has passed its gates and its review pair goes straight into the merge queue and the status line says merging is autonomous. A contributor's pull request waits for a maintainer's approval in either mode, and a fresh install starts with a person, so installing abcd changes nothing about who merges. (itd-2610150602547271)

When someone reports a security hole in your project privately, abcd keeps it private until the fix ships and then publishes it with you. `abcd embargo` brings each private advisory into a record only your machine holds, records your verdict and who gave it, checks everything you are about to make public for anything that would give an advisory away, and on release day shows the publication plan with each patched version. On your one explicit yes it publishes a single advisory first, checks it went out as planned, then publishes the rest. (itd-2610112239022766)

A person running several repositories with abcd gets, in each, a read-only report of how that repository's GitHub settings differ from a safe baseline: one line per setting, saying what it is now and what it should be. When the person says yes, abcd turns on what is missing and writes down what it changed. It never changes whether a repository is public or private, and it never requires a check the repository's own workflows do not run. (itd-2610290746077716)

Every model call abcd makes records two sides: what abcd asked for, and what actually answered. A role run through the host, a command-line runner or a provider connection names the model it requested and the settings it sent, then the model, provider, token counts and cost the answer reported; whatever the route cannot show is written down as unpinned or self-reported, never left out and never guessed. (itd-166)

> "A review came back weaker than last week's and I spent an hour blaming the diff," said a technical facilitator. (itd-166)

After a plugin update, abcd says what changed in words a person can use and offers the setup that waits. The first reply in the next session gives one line per release, printed by `abcd update news`, with the full summary one command away (`abcd update news --full`), and when the update brings something the repository needs it offers to run setup again, which asks only the new questions. Noticing the update reads only what is on disk, and nothing changes without a yes. (itd-2610161616086564, itd-2610161617096084)

Every person who commits to a repository abcd manages commits as themselves. Each person's commit identity now lives on their own machine rather than in a committed pin, so a second contributor is never told to commit under someone else's name; setup moves a committed pin to its owner's machine, and a commit made under a bot's or an AI tool's name is still refused. (itd-2610240918547728)

Setup does more of the installing for you, and asks first. When abcd needs another tool, such as a secret scanner, it installs it for the person alone, with no admin rights, into their own folder, and checks the download is the one the tool's makers published before it runs anything. When the folder abcd installs into is not on the PATH, setup asks once whether to add it to the shell settings, changes them only on a yes, and undoes the change when abcd is removed. (itd-2610240918594138, itd-2610240919030303)

When a project switches it on at setup, its release workflow builds the project's about page and development record at each release and publishes them on the project's own GitHub Pages. Every page opens with a line saying it is the record of an abcd-managed repository, and every word on it comes from the repository; the project's own workflow builds it and abcd runs nothing. (itd-2610091010348892)

Also in this release:

- abcd's typed judgements can run through a decision adapter, measured in a lab before it decides anything (itd-2609221009495079)
- A preflight waits for its turn as one whole run (itd-2610262127138979)

The line-by-line record of this release is its section in CHANGELOG.md.
