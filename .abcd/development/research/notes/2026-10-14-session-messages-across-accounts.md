# Session messages across accounts, machines and cloud sessions: options and grounds

Dated 2026-10-14. A design pass that put three options to the product
thinker for how abcd sessions tell each other things once they run under
different user accounts, on different machines and in hosted cloud sessions.
The pass was composed in the local tier and promoted here so that the second
person can read the grounds of the ruling. It folds in one adversarial security review
and one state-of-the-art pass, both held the same day. The product thinker
chose option 1 on 2026-10-15; the ruling is recorded on
[itd-2609151838312703](../../intents/drafts/itd-2609151838312703-sessions-announce-what-they-are-doing.md).

## The problem

Sessions coordinated on 2026-10-14 through the host's cross-session messages,
which reach only sessions under one operating-system account on one machine.
Five sessions exchanged dozens of them that evening, of four kinds:

- sub-agent slot loans;
- who owns a design;
- "I am merging this, rebase after";
- "I am cutting a release, hold your merge".

From 2026-10-15 a second account drains the ledger, and cloud sessions are
planned (itd-2610120738075547). Neither can receive those messages. <!-- record-lint: forward-looking -->

## Constraints already ruled

- **Coordination is visible on the shared remote or the forge.** The product
  thinker ruled this on 2026-10-03 (the decision log's entry of that day), when
  a second person began working on the repository. A local channel is an
  accelerator only, because the other person never sees it.
- **abcd never reads a file another account can write.** This ruling was
  relayed on 2026-10-14 and is not yet in the decision log. It rules out the
  shared directory that the draft
  [itd-2609151838312703](../../intents/drafts/itd-2609151838312703-sessions-announce-what-they-are-doing.md)
  first proposed.
- **Claims are branches on the shared remote.** This is
  [itd-2609091034175565](../../intents/planned/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md)
  decision 10: a hosted session's git proxy accepts pushes of branches only.
- **The repository is public.**

## Findings

- **F1: The host's own messages are per login.** They now reach cloud
  sessions and the person's other machines within one host login, through the
  host's remote connection (vendor documentation). They never reach another
  login and never show on the forge, so under the visibility ruling they stay
  an accelerator.
- **F2: Most of the traffic was locks, not conversation.**
  - "Who owns this design" is a reservation.
  - "Hold your merge" is a lease on the release.
  - "I am merging" is a notice.
  - A slot loan concerns one machine and is not cross-machine at all.

  Anthropic's sixteen-agent compiler run (Carlini, February 2026) coordinated
  with locks alone (practitioner report).
- **F3: The reservation code needs generalising to carry messages.**
  - Its fetch is fixed to its own branch prefix.
  - Its message parser requires a record id.
  - The pre-push hook exempts only the reservation prefix.
  - The live ruleset on reservation branches forbids deletion with no bypass,
    so a message prefix under the same rule could never be swept.
- **F4: A sender's identity on the remote is self-declared.** It comes from a
  commit trailer and the configured email, and anyone with push access can
  write any branch.
  - A ruleset binds a prefix to a team, a role or an app, never to one user.
  - Commits from hosted sessions are reported to arrive unverified (user
    reports, July–August 2026).
  - Which forge identity a cloud push arrives as is unverified.
- **F5: It is contested whether a hosted session may push outside its own
  branch prefix.** The host's documentation says its proxy does not limit
  branch names; older user reports say it does.
- **F6: A pattern given to `git ls-remote` still downloads every ref.** That is
  1,011 refs on 2026-10-14, 933 of them pull-request refs; `--heads` narrows it
  to branches. One call measured 0.46 s.
- **F7: There is no embargo check on the default branch yet.** One is on an
  unmerged branch due for v0.15.0. The outbound check judges only session links
  and tool footers.
- **F8: Peer text is a live prompt-injection route.** Recorded incidents:
  - the GitHub MCP toxic flow (Invariant Labs, May 2025);
  - Clinejection (February 2026);
  - Comment and Control (April 2026);
  - GitLost (July 2026).

  A fence around the text is a weak control. The architectural defence
  (CaMeL, arXiv 2503.18813; design patterns, arXiv 2506.08837) keeps
  untrusted text from reaching the agent as free text: the agent sees only
  fields abcd renders. Spotlighting (arXiv 2403.14720) lowers attack success
  but does not remove it.

## Options put to the product thinker

1. **Short fixed-form notes on the shared remote; locks stay reservations.**
   - Locks become reservations, plus a small new lease for a release cut.
   - Everything else is a notice with no free text. Each sending session owns
     one branch, and each notice is one empty-tree commit on it carrying:
     - a kind from a fixed list (merging, merged, cut started, cut ended,
       question on, answered on, handed over);
     - the record or pull request it concerns;
     - an expiry, capped for any notice that asks others to wait.
   - A question points at an item, and the conversation happens on that item.
   - Gain: it reaches every account, machine and cloud session that can push,
     with nothing new to run, and no free text means no injection route.
   - Cost: public (ids, kinds and times only), up to a minute late, idle
     sessions are not woken, and the branch list grows.
2. **One locked coordination issue on the forge.**
   - Agents post through the REST interface, which hosted sessions can reach.
   - Only comments by owners, members and collaborators are read.
   - Gain: people read and answer from a phone in full sentences, and the forge
     authenticates the author.
   - Cost: public forever with edit history, tied to one forge, and agents may
     read only a structured part.
3. **A private side repository with option 1's mechanics, plus free text.**
   - Gain: nothing is public.
   - Cost: a second repository to grant to everyone, unverified reach from
     hosted sessions, outside readers see nothing, and free text is still an
     injection route.

Rejected without a question, each the one defensible answer on the record:

- A shared machine folder: it breaks the 2026-10-14 ruling and never leaves the
  machine.
- Encryption in the public repository: metadata leaks, access cannot be
  revoked, and hosted sessions hold no key.
- Custom refs: the hosted proxy accepts branches only.
- One shared mailbox branch: concurrent pushes race.
- Any message that grants something: a message informs or asks, never
  authorises.

## Rulings that hold whichever option is picked

- The host's same-account messages stay as a wake-up accelerator only.
- **The sub-agent budget sits beside this design, not in it.**
  - The draft itd-2610100704270254 is a per-run setting in the repository's
    config.
  - The machine cap belongs to one machine and is readable from the process
    table for every account without a shared file.
  - Hosted sessions run on their own machines.
  - A capacity message would be an authorisation.
- **A failed fetch says so.** A session whose notices could not be read is told
  since when and why, never shown silence.
- **A sender is shown as unauthenticated** ("claims to be") until identity can
  be bound (F4).

## Step one of the chosen option

Two probes from a real hosted session come before anything is built:

- **P1:** push one empty commit to a branch outside the host's own prefix.
  Record whether it is accepted, and which pusher the forge's push event names:
  the person's account or an app installation. If the push is refused, the
  branch prefix goes back to the product thinker. If the pusher is an app, no
  per-person rule can bind a sender, and the cooperative model stands as
  stated for reservations.
- **P2 (option 3 only):** whether a hosted environment can be granted a second
  repository.

## Recommendation given

Option 1. Most of the traffic is locks, and the lock primitive already exists
on the remote. The remainder fits a few fixed kinds, which closes the injection
route that free text opens. A forge issue stays the place for people's prose:
the cloud intent already uses one as its fallback report channel.
