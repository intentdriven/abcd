# Research note: live co-editing of a Markdown brief that lands as one GitHub pull request (abcdesign.app)

Written for itd-2610052000424067, from the planning interview of 2026-10-14. Produced by a research agent and saved unedited; the record holds it as the agent returned it.

**Date:** 2026-10-14. Recency is judged against this date, which the request supplied. **Scope:** a small invited team edits Markdown files from one Git repository together, live and offline, with notes on passages, then submits one pull request (PR). The product wants no server-side LLM and as little hosted state as possible. **Method:** Every claim below comes from a page opened during this research and carries a tier: [EVIDENCE] (measured), [CONSENSUS] (several independent sources agree), [CONTESTED] (credible sources disagree), [DOCS] (the vendor's or maintainer's own documentation), [ANECDOTE/MARKETING] (no method behind it), or [INFERENCE] (my reasoning, not sourced). Pages that could not be opened (loro.dev returned 403) are named in prose and not linked.

---

## Summary

The simplest stack that works is to **edit the raw Markdown text, not a rich-text (WYSIWYG) document**. That means:

- **Editor and sync:** one Yjs document per session, holding one shared text per file, edited in CodeMirror 6 through `y-codemirror.next`.
- **Offline copy:** stored in the browser with `y-indexeddb`.
- **Relay:** one small relay "room" per session, which is deleted when the team submits.
- **Notes:** kept in the same Yjs document, pinned to text with Yjs relative positions.
- **Submit:** a three-way merge in the browser against the branch tip, then one commit through GitHub's API. The submitter is the author, every other member gets a `Co-authored-by:` trailer, and every member is named in the PR body.

The biggest avoidable mistake is making a WYSIWYG document the source of truth and converting it back to Markdown at Submit. GitLab measured that conversion as lossy or broken on a large share of real documents (section 1). The biggest open risk is that the relay holds a readable copy of private-repository content while a session is live (section 2).

---

## 1. CRDT and OT options

| Option | Licence | Version now | Bundle (gzipped, see caveat) | Markdown round trip |
|---|---|---|---|---|
| **Yjs** + `y-codemirror.next` (shared text) | MIT ([yjs](https://github.com/yjs/yjs); [y-codemirror.next](https://registry.npmjs.org/y-codemirror.next/latest)) | `latest` is 13.6.33; 14.0.0 only as prereleases ([npm registry](https://registry.npmjs.org/yjs)) | 20,100 B ([crdt-benchmarks](https://github.com/dmonad/crdt-benchmarks)) | Exact to the byte: the shared text *is* the Markdown file [INFERENCE] |
| **Yjs** + `y-prosemirror` (WYSIWYG) | MIT ([y-prosemirror](https://github.com/yjs/y-prosemirror)) | same | same, plus ProseMirror | Depends on a Markdown converter, which is lossy (see below) |
| **Automerge** 3 + `@automerge/prosemirror` | MIT ([automerge](https://github.com/automerge/automerge)) | 3.5.0 ([npm](https://registry.npmjs.org/@automerge/automerge/latest)); ProseMirror binding 0.2.0 ([npm](https://registry.npmjs.org/@automerge/prosemirror/latest)) | 604,118 B, measured on v2.1.10 ([crdt-benchmarks](https://github.com/dmonad/crdt-benchmarks)) | Same converter problem |
| **Loro** + `loro-prosemirror` | MIT ([loro](https://github.com/loro-dev/loro); [loro-prosemirror](https://github.com/loro-dev/loro-prosemirror)) | 1.16.4 ([npm](https://registry.npmjs.org/loro-crdt/latest)) | 399,276 B, measured on v0.10.1 ([crdt-benchmarks](https://github.com/dmonad/crdt-benchmarks)) | Same converter problem |
| **CodeMirror OT** (operational transformation, Haverbeke's design) | not checked | not checked | small | Exact (plain text) |
| **Peritext** | research prototype | none | not applicable | Inline formatting only |

**Findings**

- **WYSIWYG-to-Markdown conversion loses or breaks formatting at scale. [EVIDENCE]** A GitLab merge request (opened 2026-09-08) tested the rich-text editor's Markdown output on 1,277 editor documents. Before the fix, 692 rendered correctly, 154 were lossy and 423 were broken. After the fix: 1,072 correct, 112 lossy, 84 broken. On 735 real issue descriptions, 163 saved differently ([GitLab MR 254236](https://gitlab.com/gitlab-org/gitlab/-/merge_requests/254236)). That is GitLab's own ProseMirror-based converter, tuned by a large team, and it is still not lossless.
  - Tiptap's official Markdown extension calls itself "a early release … may have edge cases". It says "Comments are not supported yet" and allows only one child node per table cell ([Tiptap Markdown docs](https://tiptap.dev/docs/editor/markdown)). [DOCS]
  - The upstream `prosemirror-markdown` covers only a basic schema: there are no table or strikethrough entries. Its GitHub repository was archived on 2026-04-01 and moved to code.haverbeke.berlin ([prosemirror-markdown](https://github.com/ProseMirror/prosemirror-markdown)). [DOCS]
  - **Consequence for abcdesign:** a WYSIWYG source of truth turns every Submit into a diff that rewrites lines nobody touched. Editing the raw text avoids the whole class of problem. [INFERENCE, from the evidence above]
- **Yjs has the widest set of ready-made integrations. [CONSENSUS]** Its README lists bindings for ProseMirror, CodeMirror, Monaco, Tiptap, Lexical, BlockNote, Milkdown and others. It also lists providers for WebRTC, WebSocket and IndexedDB ([yjs](https://github.com/yjs/yjs)). Its docs name Liveblocks, Y-Sweet and Tiptap as hosted Yjs services ([docs.yjs.dev](https://docs.yjs.dev/)).
- **Yjs is mid-way through a major version change. [DOCS]** The `main` branches of `y-codemirror.next` and `y-prosemirror` now target unstable Yjs v14 (`@y/y`, `@y/codemirror`, `@y/prosemirror`). Most users are told to stay on the stable v13 packages ([y-codemirror.next](https://github.com/yjs/y-codemirror.next); [y-prosemirror](https://github.com/yjs/y-prosemirror)). The npm `latest` tag is still 13.6.33 ([registry](https://registry.npmjs.org/yjs)).
- **Automerge 3 cut memory use sharply, but the evidence is one example. [ANECDOTE]** It released in July 2025 and claims ">10x" less memory. The only test given is pasting Moby Dick: 700 MB in Automerge 2 against 1.3 MB in Automerge 3 ([Automerge 3.0](https://automerge.org/blog/automerge-3/)). The ProseMirror binding describes itself as "beta quality software. The API will probably change" ([README](https://cdn.jsdelivr.net/npm/@automerge/prosemirror@0.2.0/README.md)). [DOCS] Rich text uses inline "marks" for formatting and "block markers" for structure, and `updateSpans` "does not yet update the formatting spans" ([rich text docs](https://automerge.org/docs/reference/documents/rich-text/)). [DOCS]
- **Loro has native rich text and the Fugue text algorithm, but its integrations are thinner.** Its repo lists a rich-text CRDT, "text editing with Fugue" and announces "Loro 1.0 is out!" ([loro](https://github.com/loro-dev/loro)). The 1.0 announcement on loro.dev returned 403 and was not read. `loro-prosemirror` states no stability level; at the time of reading it had 99 commits and 144 stars ([loro-prosemirror](https://github.com/loro-dev/loro-prosemirror)). [DOCS]
- **Benchmarks are disputed and stale. [CONTESTED]**
  - The standard bundle and memory benchmark repository is maintained by the Yjs author (Kevin Jahns). It tests old versions: loro 0.10.1 and automerge 2.1.10 ([crdt-benchmarks](https://github.com/dmonad/crdt-benchmarks)).
  - Loro's author (May 2024) said the Loro numbers "are not stable" while its encoding format was changing. He also argued Yjs should be benchmarked with garbage collection off for fairness, and admitted "the WASM size is larger" ([Yjs forum](https://discuss.yjs.dev/t/yjs-vs-loro-new-crdt-lib/2567/4)).
  - Treat the bundle figures as rough size classes: about 20 KB for Yjs against hundreds of KB for the WASM libraries. They are not current measurements.
- **History is not a reason to choose Automerge or Loro here. [INFERENCE]** Their main advantage is full editing history and "time travel" ([Yjs forum](https://discuss.yjs.dev/t/yjs-vs-loro-new-crdt-lib/2567/4)). abcdesign already has Git as its history, and a session is short-lived.
- **Peritext is the foundation, not a product. [DOCS]** It is a 2021 Ink & Switch prototype for inline formatting, including overlapping comments that don't grow when text is typed at their edge. It does not handle blocks (headings, lists, tables) ([Peritext](https://inkandswitch.com/peritext/)). Automerge's rich text descends from this work.
- **Concurrent inserts at the same spot can interleave. [EVIDENCE]** When two people type at the same position, "most" replicated-list algorithms can mix their passages together, in both CRDTs and OT. Fugue (Weidner and Kleppmann, IEEE TPDS, Nov 2025) minimises this ([Kleppmann](https://martin.kleppmann.com/2025/11/01/fugue-minimizing-interleaving.html)). The page does not say whether Yjs is affected.
- **Eg-walker is a related research algorithm. [EVIDENCE]** It claims an order of magnitude less steady-state memory than CRDTs and much faster merging of long-diverged branches than OT (EuroSys 2025). Its implementations are diamond-types and a TypeScript reference ([Kleppmann](https://martin.kleppmann.com/2025/03/30/eg-walker-collaborative-text.html)). It is not a mainstream editor stack.

## 2. Sync and relay infrastructure

| Option | Where the shared copy lives during a session | Cost | Privacy for a private repo | Status |
|---|---|---|---|---|
| **y-webrtc** (peer-to-peer) | Only in the browsers that are online | Signalling servers only | Content goes browser to browser; a `password` encrypts signalling ([y-webrtc](https://github.com/yjs/y-webrtc)) | Maintained [DOCS] |
| **Cloudflare Durable Object + y-partyserver** | One Durable Object (DO) per room. In-memory only unless you implement `onLoad`/`onSave` ([y-partyserver](https://github.com/cloudflare/partykit/tree/main/packages/y-partyserver)) | Free plan includes DOs with SQLite storage; hibernating objects are not billed for time ([DO pricing](https://developers.cloudflare.com/durable-objects/platform/pricing/)) | Server holds readable document state (it calls `Y.applyUpdate`) | ISC licence ([cloudflare/partykit](https://github.com/cloudflare/partykit)) |
| **Hocuspocus** | Your Node, Bun, Deno or Workers process, with SQLite and other extensions | Self-host | Server holds readable state | MIT; Workers supported since v4 ([overview](https://tiptap.dev/docs/hocuspocus/getting-started/overview); [repo](https://github.com/ueberdosis/hocuspocus)) |
| **Liveblocks** (hosted) | Liveblocks: AWS us-east-1 by default, realtime on Cloudflare ([data storage](https://liveblocks.io/docs/platform/data-storage)) | Free tier: 10 simultaneous connections per room, 3,000 realtime minutes, 200 comments ([pricing](https://liveblocks.io/pricing)) | A third party holds readable content; region lock and self-hosting are Enterprise-only | Server source is AGPL v3 and "not quite yet" ready for production self-hosting ([blog](https://liveblocks.io/blog/open-sourcing-the-liveblocks-sync-engine-and-dev-server)) |
| **y-sweet** | S3-compatible storage or filesystem; in-memory by default ([y-sweet](https://github.com/jamsocket/y-sweet)) | Self-host only now | Server holds readable state | See the warning below |

**Findings**

- **Peer-to-peer alone can't satisfy "keep working offline" for a team. [INFERENCE, from the README]** The y-webrtc README describes no storage. Clients sync only if they are connected to each other, directly or through other peers ([y-webrtc](https://github.com/yjs/y-webrtc)). So if the person holding the newest edits is offline, nobody else can get them. The default signalling list is `wss://signaling.yjs.dev` plus two Heroku hosts, and the README makes no reliability claim for them. [DOCS]
- **A per-session Durable Object is the cheapest relay that matches "minimal hosted state". [DOCS + INFERENCE]**
  - Pricing facts: the free plan includes 100,000 requests a day and 5 GB of stored data. Incoming WebSocket messages count 20:1 toward requests. Cloudflare's own example of "WebSocket Hibernation with light traffic" (100 objects, 100 connections each) costs about $20.65 a month on the paid plan ([DO pricing](https://developers.cloudflare.com/durable-objects/platform/pricing/)).
  - My estimate is that a handful of small teams would fit in the free plan. That is untested.
  - The same Worker can also do the GitHub token exchange. GitHub's code exchange needs the app's `client_secret` ([GitHub App user tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app)), so some server-side step is unavoidable.
- **y-sweet: don't adopt. [EVIDENCE: observed during this research]**
  - Jamsocket joined Modal on 2025-07-10 ([Modal blog](https://modal.com/blog/jamsocket-is-joining-modal)).
  - Fetching `docs.jamsocket.com/y-sweet/...` now redirects to that Modal post, and `jamsocket.xyz` redirects to a GoDaddy domain-for-sale page.
  - The newest commit on the open-source repo is dated 2025-12-04 ([commits](https://github.com/jamsocket/y-sweet/commits/main)), about 10 months before this note.
  - Obsidian's Relay plugin runs on a fork of y-sweet (MIT) with a proprietary login, permissions and billing service ([Relay](https://github.com/No-Instructions/Relay)). That shows y-sweet works, but the original project has no visible owner.
- **Privacy: every standard Yjs server can read the document. [DOCS]** y-partyserver's own example loads the whole document into the server with `Y.applyUpdate` and saves it with `encodeStateAsUpdate` ([y-partyserver](https://github.com/cloudflare/partykit/tree/main/packages/y-partyserver)). For a private repository, the relay operator — abcdesign itself, or Liveblocks — holds readable content while a session is live. End-to-end encrypted options exist but are not mature:
  - **secsync** (Apache-2.0) relays encrypted Yjs and Automerge updates and labels itself "beta software". The server still sees document IDs, client public keys, who has access and when, and roughly how much each person contributes. It "could remove a participant without detection" ([secsync](https://github.com/serenity-kit/secsync)).
  - **Automerge's Keyhive integration (ARK)** says it is "still in alpha" ([ARK guide](https://automerge.org/docs/keyhive/ark-api-guide/)).
- **Offline storage in the browser is reliable except on Safari. [DOCS]** `y-indexeddb` stores the document in the browser so editing works offline ([y-indexeddb](https://github.com/yjs/y-indexeddb)). Browsers evict best-effort storage under disk pressure, least-recently-used first, unless `navigator.storage.persist()` was granted. Safari, with cross-site tracking prevention on, deletes script-written data after seven days of browser use without a click or tap on the site ([MDN](https://developer.mozilla.org/en-US/docs/Web/API/Storage_API/Storage_quotas_and_eviction_criteria)).
- **Contrarian view: you may not need a CRDT, but not for this product. [CONSENSUS]** Zak Knill argues most collaborative apps can use presence, locking and small updates instead. He explicitly excludes offline-first apps and text editing ([knill, 2023](https://zknill.io/posts/collaboration-no-crdts/)). Haverbeke's CodeMirror OT design needs a central server to order changes, and does not address offline divergence ([Haverbeke, 2020](https://marijnhaverbeke.nl/blog/collaborative-editing-cm.html)). abcdesign needs both offline work and text editing, so a CRDT is justified.

## 3. Notes anchored to passages that survive concurrent edits

- **Yjs relative positions are the standard tool. [DOCS]** A relative position points at a specific character rather than a numeric offset, so other people's edits don't move it. All peers resolve it to the same place once synced. The docs name "anchoring a comment to a range of text" (two positions) as a use. It resolves to `null` if its container was deleted ([relative positions](https://docs.yjs.dev/api/relative-positions)). `y-prosemirror` recommends the same for comments and returns `null` rather than a nearby position when content is deleted along with its parent ([y-prosemirror](https://github.com/yjs/y-prosemirror)). What happens when only the anchored characters are deleted was not confirmed in this research; test it.
- **Plain OT position-mapping can't anchor notes reliably. [DOCS]** Haverbeke writes that, without CRDT-style IDs, position mapping "can't converge", so two users can see the same note in different places ([Haverbeke](https://marijnhaverbeke.nl/blog/collaborative-editing-cm.html)).
- **Automerge has an equivalent.** `getCursor` gives a stable anchor with a direction setting for deleted characters ([automerge-repo getCursor, found via search, not opened]). Peritext models comments as overlapping marks that don't grow when text is typed at their edge ([Peritext](https://inkandswitch.com/peritext/)). [DOCS]
- **For notes that must outlive the session, also store a text-quote anchor. [DOCS]** The W3C Web Annotation Recommendation (2017) defines a `TextQuoteSelector`: the quoted text (`exact`) plus some text before it (`prefix`) and after it (`suffix`). It calls offset-based position selectors "very brittle" ([W3C](https://www.w3.org/TR/annotation-model/#text-quote-selector)). Storing both the Yjs anchors (for the live session) and a quote anchor (to re-find the passage in the plain file after Submit) is a cheap belt-and-braces. [INFERENCE]
- **Mapping notes onto the PR is limited. [DOCS]** PR review comments attach to a `line` of the PR *diff*, or to a whole file with `subject_type: file`. Creating them quickly can trigger secondary rate limits ([PR review comments](https://docs.github.com/en/rest/pulls/comments#create-a-review-comment-for-a-pull-request)). Notes on unchanged passages therefore need either a file-level comment or a list in the PR body.
- **Ready-made comment products are paid and hosted. [DOCS]** Tiptap Comments is a Pro extension that needs Tiptap Cloud ([Tiptap Comments](https://tiptap.dev/docs/comments/getting-started/overview)). Liveblocks charges $0.01 per comment on Pro and Team after credits ([pricing](https://liveblocks.io/pricing)).

## 4. From the shared document to one clean PR with several co-authors

- **Load each file into the shared document exactly once. [CONSENSUS]** The Yjs maintainer has said, repeatedly from 2021 to 2022, that if every client inserts the same starting content, the content appears twice. His advice is to initialise once, or to apply one pre-encoded update everywhere, and "Never manipulate clientIDs" ([Yjs forum](https://discuss.yjs.dev/t/initial-offline-value-of-a-shared-document/465)). Moriz Büsing reports the same pitfall and fix independently ([Büsing, 2022](https://morizbuesing.com/blog/initializing-a-yjs-document-with-a-common-value)). For abcdesign: the starter loads each file's Git content at a recorded base commit (SHA) before inviting anyone, and stores that SHA in the document.
- **Committing through GitHub's API. [DOCS]**
  - `createCommitOnBranch` (GraphQL) "appends a commit … as the authenticated user". Its required `expectedHeadOid` is the commit the branch head is expected to be at. Commits are signed by GitHub "when supported". It "does not support specifying the author or committer"; the lower-level Git Database REST API does ([GraphQL commits](https://docs.github.com/en/graphql/reference/commits)).
  - The branch is created with `POST /git/refs` ([refs](https://docs.github.com/en/rest/git/refs#create-a-reference)). The PR is opened with `POST /pulls`. Draft PRs on private repos need GitHub Team or Enterprise Cloud ([pulls](https://docs.github.com/en/rest/pulls/pulls#create-a-pull-request)).
- **Crediting every member. [DOCS]** Add one `Co-authored-by: name <email>` line per member. Credit counts only if the email is linked to the member's GitHub account ([co-authors](https://docs.github.com/en/pull-requests/committing-changes-to-your-project/creating-and-editing-commits/creating-a-commit-with-multiple-authors)). Use the `ID+USERNAME@users.noreply.github.com` form, which survives username changes; the older `USERNAME@` form does not ([email reference](https://docs.github.com/en/account-and-profile/reference/email-addresses-reference)). So the commit is authored by whoever presses Submit, with everyone else as co-authors, and the PR body also names every member.
- **When the repo changes during the session.** Two options:
  - **Simplest:** branch from the session's base commit and open the PR. GitHub then flags any conflicts. Its web conflict editor handles only "simple competing line change conflicts", and resolving there merges the whole base branch into the PR branch ([resolving conflicts](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/addressing-merge-conflicts/resolving-a-merge-conflict-on-github)). [DOCS]
  - **Better:** at Submit, for each file whose content changed upstream, run a three-way merge in the browser: base version, current tip, session result. `node-diff3` (MIT, runs in the browser) has `diff3Merge` and `mergeDiff3` with Git-style conflict markers ([node-diff3](https://github.com/bhousel/node-diff3)). [DOCS] It splits strings on whitespace by default; pass a line separator for line-level merges. Clean merges commit on top of the tip, with `expectedHeadOid` guarding against a race. Conflicts are shown to the whole group in the shared editor before Submit is allowed. [INFERENCE]
- **Repository access. [DOCS]** GitHub App user tokens get only the overlap of what the app and the user can access, last 8 hours (refresh tokens 6 months), and support PKCE ([user tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app)). Membership can be checked with `GET /collaborators/{username}/permission` ([collaborators](https://docs.github.com/en/rest/collaborators/collaborators#get-repository-permissions-for-a-user)). A member with read-only access cannot be the one who submits. [INFERENCE]

## 5. Prior art: "edit in the browser, submit as a PR"

None of the following documents a live multi-person session that ends in one PR.

- **github.dev:** one person in the browser; commit and push to a branch, then create a PR. No real-time collaboration is described ([github.dev](https://docs.github.com/en/codespaces/the-githubdev-web-based-editor)). [DOCS]
- **Decap CMS editorial workflow:** each draft gets a branch (`cms/collection/slug`) and a PR; publishing merges it. Simultaneous editing and locking are not covered ([Decap](https://decapcms.org/docs/editorial-workflows/)). [DOCS]
- **TinaCMS:** saving to a protected branch creates a new branch and a draft PR. Available on "select paid plans"; no real-time multi-user editing is described ([Tina](https://tina.io/docs/tinacloud/editorial-workflow)). [DOCS]
- **Keystatic GitHub mode:** a GitHub App with a branch picker; collaborators need write access. PRs and concurrent editing are not documented ([Keystatic](https://keystatic.com/docs/github-mode)). [DOCS]
- **HackMD GitHub sync:** pushes named versions as commits and pulls branch changes. HackMD notes it does not lock a note during a merge. PR creation and commit authorship are not documented ([HackMD](https://hackmd.io/s/link-with-github)). [DOCS]
- **Obsidian Relay:** real-time editing of Markdown over Yjs and a fork of y-sweet, with an optional self-hosted relay. No Git integration is described ([Relay](https://github.com/No-Instructions/Relay)). [DOCS]
- **Reading:** the "multiplayer session → one PR naming everyone" combination looks unclaimed. That means abcdesign has no proven reference design to copy. [INFERENCE from absence in docs]

---

## Ranked recommendations (simplest viable stack first)

1. **Edit raw Markdown text with Yjs v13 and `y-codemirror.next`.** Use one Yjs document per session, one shared text per file, and pin Yjs 13.6.x. This makes the round trip byte-exact, gives the smallest bundle, and uses the most-used bindings. Do not take `@y/*` v14 until it is stable. — [GitLab MR](https://gitlab.com/gitlab-org/gitlab/-/merge_requests/254236), [yjs](https://github.com/yjs/yjs), [y-codemirror.next](https://github.com/yjs/y-codemirror.next). [EVIDENCE for the problem being avoided; DOCS for the stack]
2. **Load content once, from a recorded base commit.** The starter loads each file before invites go out, and the base SHA is stored in the document. Never let joiners re-load. — [Yjs forum](https://discuss.yjs.dev/t/initial-offline-value-of-a-shared-document/465). [CONSENSUS]
3. **Offline: `y-indexeddb` plus `navigator.storage.persist()`.** Warn Safari users about the seven-day eviction, and encourage Submit before people go idle. — [y-indexeddb](https://github.com/yjs/y-indexeddb), [MDN](https://developer.mozilla.org/en-US/docs/Web/API/Storage_API/Storage_quotas_and_eviction_criteria). [DOCS]
4. **Relay: one Cloudflare Worker with one Durable Object per session using `y-partyserver`.** Save into the Durable Object's SQLite storage, and delete the room on Submit or after a time-to-live. The same Worker does the GitHub token exchange, and should hold nothing else. Hocuspocus on Workers is the MIT alternative. — [DO pricing](https://developers.cloudflare.com/durable-objects/platform/pricing/), [y-partyserver](https://github.com/cloudflare/partykit/tree/main/packages/y-partyserver), [Hocuspocus](https://tiptap.dev/docs/hocuspocus/getting-started/overview). [DOCS]
5. **Notes go in the same document.** Store them as a map of note id → {two relative positions, a W3C-style text-quote anchor, author, text}. At Submit, list them in the PR body; optionally post the ones on changed lines as PR review comments, and the rest as file-level comments. — [Yjs relative positions](https://docs.yjs.dev/api/relative-positions), [W3C](https://www.w3.org/TR/annotation-model/#text-quote-selector), [PR review comments](https://docs.github.com/en/rest/pulls/comments#create-a-review-comment-for-a-pull-request). [DOCS]
6. **Submit: three-way merge in the browser, then commit and open the PR.** Steps: fetch the tip; run a line-level diff3 merge per changed file; if clean, create the branch at the tip, call `createCommitOnBranch` with `expectedHeadOid`, add `Co-authored-by:` trailers using `ID+USERNAME` noreply emails, and open the PR naming all members. If there are conflicts, resolve them together in the editor first. The fallback is to branch from the base commit and let GitHub flag conflicts. — [GraphQL commits](https://docs.github.com/en/graphql/reference/commits), [co-authors](https://docs.github.com/en/pull-requests/committing-changes-to-your-project/creating-and-editing-commits/creating-a-commit-with-multiple-authors), [node-diff3](https://github.com/bhousel/node-diff3). [DOCS]
7. **Auth: a GitHub App with user tokens (PKCE).** Check each invitee's permission on the repository. Only a member with write access can submit. — [user tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app), [collaborators](https://docs.github.com/en/rest/collaborators/collaborators#get-repository-permissions-for-a-user). [DOCS]
8. **Later, if private-repository customers ask: end-to-end encryption.** The relay would store and forward only encrypted Yjs updates (secsync's design), with the group key passed in the invite. Don't build this first: secsync is beta and still exposes metadata. — [secsync](https://github.com/serenity-kit/secsync). [DOCS]

## Not worth adopting

- **A WYSIWYG editor (ProseMirror, Tiptap, Milkdown) as the source of truth with Markdown generated at Submit.** Measured as lossy or broken on a large share of documents; Tiptap's converter is an "early release" without comment support ([GitLab MR](https://gitlab.com/gitlab-org/gitlab/-/merge_requests/254236); [Tiptap](https://tiptap.dev/docs/editor/markdown)). If a friendlier view is wanted, style the CodeMirror Markdown view instead. [INFERENCE]
- **Liveblocks.** A third party would hold readable private-repo content, us-east-1 by default. Region lock and self-hosting are Enterprise-only, and the open-source server is AGPL and not ready for production ([data storage](https://liveblocks.io/docs/platform/data-storage); [blog](https://liveblocks.io/blog/open-sourcing-the-liveblocks-sync-engine-and-dev-server)). This contradicts "minimal hosted state".
- **y-sweet.** The hosted service is gone, the vendor's domain is parked, and the last open-source commit was 2025-12-04 ([Modal](https://modal.com/blog/jamsocket-is-joining-modal); [commits](https://github.com/jamsocket/y-sweet/commits/main)).
- **Automerge or Loro for this product.** Their ProseMirror bindings are beta or of unstated maturity, their bundles are an order of magnitude larger on the (stale) benchmarks, and their history features duplicate Git ([@automerge/prosemirror](https://cdn.jsdelivr.net/npm/@automerge/prosemirror@0.2.0/README.md); [crdt-benchmarks](https://github.com/dmonad/crdt-benchmarks)). [CONTESTED on performance; the benchmark owner is the Yjs author]
- **CodeMirror's OT design.** It needs a central server to order changes, does not address offline divergence, and its note positions can disagree between users ([Haverbeke](https://marijnhaverbeke.nl/blog/collaborative-editing-cm.html)).
- **Pure peer-to-peer (y-webrtc only).** No shared copy exists when the person with the newest edits is offline ([y-webrtc](https://github.com/yjs/y-webrtc)).
- **Tiptap Comments.** Paid, and tied to Tiptap Cloud ([Tiptap Comments](https://tiptap.dev/docs/comments/getting-started/overview)).
- **Locking instead of a CRDT.** Its own advocate excludes text editing and offline use ([knill](https://zknill.io/posts/collaboration-no-crdts/)).
- **Using Peritext, Keyhive or Eg-walker directly.** These are a research prototype, an alpha and a research algorithm respectively ([Peritext](https://inkandswitch.com/peritext/); [ARK](https://automerge.org/docs/keyhive/ark-api-guide/); [Eg-walker](https://martin.kleppmann.com/2025/03/30/eg-walker-collaborative-text.html)).

## Open risks

1. **The relay can read private-repository content.** Every standard Yjs server applies updates in readable form. Mitigations: short-lived rooms, deletion on Submit, and stating this plainly at invite time. End-to-end encryption is beta at best ([y-partyserver](https://github.com/cloudflare/partykit/tree/main/packages/y-partyserver); [secsync](https://github.com/serenity-kit/secsync)).
2. **The Yjs v13 to v14 migration.** v14 packages are renamed to `@y/*`. A third-party mirror of the first v14 prerelease notes says v13 can't read v14 updates if content has been moved; I did not confirm this on GitHub ([y-codemirror.next](https://github.com/yjs/y-codemirror.next); [npm registry](https://registry.npmjs.org/yjs)).
3. **Long offline divergence.** Merges always converge, but concurrent inserts at the same spot can interleave into unreadable text ([Fugue](https://martin.kleppmann.com/2025/11/01/fugue-minimizing-interleaving.html)). The PR diff review is the human safety net.
4. **Browser eviction.** A user who only ever edited offline can lose unsynced edits to Safari's seven-day rule or to storage pressure ([MDN](https://developer.mozilla.org/en-US/docs/Web/API/Storage_API/Storage_quotas_and_eviction_criteria)).
5. **GitHub App installation.** User tokens cover only what both the app and the user can reach, so the app must be installed on the repo. That may need an org owner's approval ([user tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app)). [INFERENCE about the approval step]
6. **Authorship limits.** `createCommitOnBranch` always makes the submitter the author, and signing is only "when supported". Co-author credit depends on the email being linked to each member's account ([GraphQL commits](https://docs.github.com/en/graphql/reference/commits); [co-authors](https://docs.github.com/en/pull-requests/committing-changes-to-your-project/creating-and-editing-commits/creating-a-commit-with-multiple-authors)).
7. **No proven reference design.** No surveyed tool documents this exact flow, so the multi-person Submit path is new ground (section 5).
8. **Untested note behaviour.** What a relative position does when only its anchored characters are deleted was not confirmed in this research. Test it before shipping notes.

## Sources

- https://gitlab.com/gitlab-org/gitlab/-/merge_requests/254236
- https://github.com/ProseMirror/prosemirror-markdown
- https://tiptap.dev/docs/editor/markdown
- https://tiptap.dev/docs/comments/getting-started/overview
- https://github.com/yjs/yjs
- https://docs.yjs.dev/
- https://docs.yjs.dev/api/relative-positions
- https://registry.npmjs.org/yjs
- https://github.com/yjs/y-codemirror.next
- https://registry.npmjs.org/y-codemirror.next/latest
- https://github.com/yjs/y-prosemirror
- https://github.com/yjs/y-webrtc
- https://github.com/yjs/y-indexeddb
- https://github.com/dmonad/crdt-benchmarks
- https://discuss.yjs.dev/t/yjs-vs-loro-new-crdt-lib/2567/4
- https://discuss.yjs.dev/t/initial-offline-value-of-a-shared-document/465
- https://morizbuesing.com/blog/initializing-a-yjs-document-with-a-common-value
- https://automerge.org/blog/automerge-3/
- https://automerge.org/docs/reference/documents/rich-text/
- https://github.com/automerge/automerge
- https://registry.npmjs.org/@automerge/automerge/latest
- https://registry.npmjs.org/@automerge/prosemirror/latest
- https://cdn.jsdelivr.net/npm/@automerge/prosemirror@0.2.0/README.md
- https://automerge.org/docs/keyhive/ark-api-guide/
- https://github.com/loro-dev/loro
- https://github.com/loro-dev/loro-prosemirror
- https://registry.npmjs.org/loro-crdt/latest
- https://inkandswitch.com/peritext/
- https://martin.kleppmann.com/2025/11/01/fugue-minimizing-interleaving.html
- https://martin.kleppmann.com/2025/03/30/eg-walker-collaborative-text.html
- https://marijnhaverbeke.nl/blog/collaborative-editing-cm.html
- https://zknill.io/posts/collaboration-no-crdts/
- https://github.com/cloudflare/partykit
- https://github.com/cloudflare/partykit/tree/main/packages/y-partyserver
- https://developers.cloudflare.com/durable-objects/platform/pricing/
- https://tiptap.dev/docs/hocuspocus/getting-started/overview
- https://github.com/ueberdosis/hocuspocus
- https://liveblocks.io/pricing
- https://liveblocks.io/docs/platform/data-storage
- https://liveblocks.io/blog/open-sourcing-the-liveblocks-sync-engine-and-dev-server
- https://github.com/jamsocket/y-sweet
- https://github.com/jamsocket/y-sweet/commits/main
- https://modal.com/blog/jamsocket-is-joining-modal
- https://github.com/No-Instructions/Relay
- https://github.com/serenity-kit/secsync
- https://developer.mozilla.org/en-US/docs/Web/API/Storage_API/Storage_quotas_and_eviction_criteria
- https://www.w3.org/TR/annotation-model/#text-quote-selector
- https://docs.github.com/en/pull-requests/committing-changes-to-your-project/creating-and-editing-commits/creating-a-commit-with-multiple-authors
- https://docs.github.com/en/account-and-profile/reference/email-addresses-reference
- https://docs.github.com/en/graphql/reference/commits
- https://docs.github.com/en/rest/git/refs#create-a-reference
- https://docs.github.com/en/rest/pulls/pulls#create-a-pull-request
- https://docs.github.com/en/rest/pulls/comments#create-a-review-comment-for-a-pull-request
- https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/addressing-merge-conflicts/resolving-a-merge-conflict-on-github
- https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app
- https://docs.github.com/en/rest/collaborators/collaborators#get-repository-permissions-for-a-user
- https://docs.github.com/en/codespaces/the-githubdev-web-based-editor
- https://github.com/bhousel/node-diff3
- https://decapcms.org/docs/editorial-workflows/
- https://tina.io/docs/tinacloud/editorial-workflow
- https://keystatic.com/docs/github-mode
- https://hackmd.io/s/link-with-github

**Not read:** loro.dev (blog and about pages returned 403), so the Loro 1.0 announcement date is not given. Automerge's `getCursor` reference appeared only in search results and was not opened; it is cited without a link.
