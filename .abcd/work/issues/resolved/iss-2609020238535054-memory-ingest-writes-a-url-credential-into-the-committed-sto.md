---
schema_version: 1
id: "iss-2609020238535054"
slug: "memory-ingest-writes-a-url-credential-into-the-committed-sto"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/ingest.go"
resolution: "sanitisedOrigin clears basic-auth userinfo and drops the credential-shaped query keys (token, api_key, apikey, access_token, password, secret; case-insensitive) where the fetched final URL becomes the stored origin and title, after the hidden-rune encoding so a URL carrying a control byte still parses; every fetch-failure message, the byte-cap refusal and the redirect-count refusal now render through redactedSource, which masks the password and drops the same query keys, with a textual fallback for a string url.Parse refuses. TestIngestStripsURLCredential pins all four arms: the userinfo never reaches any file under .abcd/memory or .sources_index.json nor the marshalled IngestResult, the token= and Api_Key= parameters are dropped from the stored origin while q=keep survives, and neither the transport failure nor the byte-cap refusal echoes the password. Completed in a later commit on the same branch: filtering the URL argument was not enough, because net/http wraps every client error in a *url.Error whose message re-prints the whole request URL and masks only the basic-auth password — so the %v of the caller's own message put a ?token= credential straight back. transportCause unwraps to the innermost non-*url.Error cause at every fetch-failure site, an IngestError a client wrapped is returned unwrapped rather than through its wrapper, and the redirect refusal and the plaintext refusal at the connection render through redactedSource instead of URL.Redacted, which masks userinfo but not the query. Three further arms of TestIngestStripsURLCredential pin it. The key list was also narrowed in that round: key and signature name a document section or a content signature at least as often as a secret, and dropping them truncated an address the citation exists to reproduce, so they came off — a fourth arm pins that an addressing ?key= survives while the bearer token beside it does not."
impact: fix
---

memory ingest writes a URL credential into the committed store: materialFromFetched sets both the page origin and the page title from the fetched final URL verbatim, so basic-auth userinfo (user:pass@) and credential-shaped query keys survive into the page frontmatter's source.citation, into .sources_index.json (origin and consumers.memory.citation) and into the IngestResult.Citation the --json render prints. redactText cannot see an opaque basic-auth password, so the scanner backstop does not catch it. The six 'fetch failed for %s' sites in acquireSource, defaultFetch and readFetchedResponse echo the raw source the same way, as do the byte-cap refusal and the redirect-count refusal. CWE-522/CWE-532. The fix must clear the userinfo and drop the credential-shaped query keys (token, api_key, apikey, access_token, password, secret; case-insensitive) where the final URL becomes the stored origin and title, and route every fetch-failure message through the redacting renderer.

## Sibling sweep

Every site in `internal/core/memory` that interpolates a fetched URL into a
message or a stored field is fixed here: the six `fetch failed for %s` sites,
the byte-cap refusal, the redirect-count refusal, and the origin/title
assignment in `materialFromFetched`. The scheme refusals already rendered
through `redactedSource` / `url.URL.Redacted`.

Out of scope, named rather than fixed:

- `internal/core/cite/fetch.go` echoes the checked URL into `CheckOutcome.URL`
  and its messages. Its input is a citation already committed to a document
  rather than an operator-typed source, so it is a different trust boundary;
  the same file's unpinned redirect scheme is already recorded as
  iss-2609012037440084.
- `internal/core/update/update.go` interpolates a release tag, not a
  caller-supplied URL, and already masks a redirect hop with `Redacted()`.

## Grounds

- pursued: a stored origin is a citation, not a way back in, so the credential is dropped structurally by name rather than left to a scanner that cannot see an opaque password; the address is otherwise reproduced byte-identical, and a citation that lost a legitimate query parameter would show the key list is too wide
