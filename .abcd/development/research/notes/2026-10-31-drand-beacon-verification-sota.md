# Verifying a drand beacon in Go without a dependency tree: a state-of-the-art check

Date: 2026-10-31. Asked for at the planning interview of itd-2610301750162657,
where the product thinker ruled that abcd verifies the beacon's signature
itself at pick time and signed off the one dependency that needs. Recency was
judged against this date. Evidence tiers: **A** a published standard or the
primary source itself (a project's own code, release notes or live API),
**B** an audit report or a rigorous industry account, **C** a third-party
restatement. Every link below was opened on that date, except where a line
says otherwise. Not done: running the test vector through any candidate
library; the first run of the spec's `TestBeaconVerify` is that check.

## The question

The spot-check (spc-2610301925429250) seeds its draw from drand's quicknet
beacon. A reply carries a round and a BLS signature; the randomness is the
SHA-256 of the signature bytes. Without verifying the signature, abcd would
trust whatever `api.drand.sh` or anything between it and abcd answered, and
the one property the beacon exists for, that no agent can steer the draw,
would rest on the transport alone. Which Go library verifies that signature
with the smallest footprint, under a licence abcd can carry, with no cgo, and
how exactly is the check shaped?

## What the evidence says

1. **The chain's parameters, from its own `info` reply** (tier A). Quicknet,
   chain hash `52db9ba70e0cc0f6eaf7803dd07447a1f5477735fd3f661792ba94600c84e971`,
   answers `schemeID bls-unchained-g1-rfc9380`, `period 3`, `genesis_time
   1692803367`, and the group public key, a 96-byte compressed G2 point:
   `83cf0f2896adee7eb8b5f01fcad3912212c437e0073e911fb90022d3e760183c8c4b450b6a0a6c3ac6a5776a2d1064510d1fec758c921cc22b0e17e63aaf4bcb5ed66304de9cf809bd274ca73bab4af5a6e9c76a4bc09e76eae8991ef5ece45a`
   (<https://api.drand.sh/52db9ba70e0cc0f6eaf7803dd07447a1f5477735fd3f661792ba94600c84e971/info>).
   Both are pinned as constants in the spec, so a reply from any other chain
   is refused before verification is tried.
2. **The scheme, from drand's source** (tier A). In `crypto/schemes.go`, the
   unchained G1 scheme puts signatures on G1 and keys on G2, signs the SHA-256
   of the round as a big-endian uint64, and uses the domain separation tag
   `BLS_SIG_BLS12381G1_XMD:SHA-256_SSWU_RO_NUL_`, the RFC 9380 default for
   G1 (<https://github.com/drand/drand/blob/master/crypto/schemes.go>).
   drand's own post on retiring fastnet says quicknet was started so that the
   chain signs under the correct tag
   (<https://docs.drand.love/blog/fastnet-to-be-sunset>), so quicknet is the
   chain whose signatures a standard-conforming verifier accepts.
3. **cloudflare/circl `sign/bls` fits the shape with three indirect modules**
   (tier A for the code and release notes). The package is pure Go under
   BSD-3-Clause; the module's `go.mod` requires only `go-ristretto`,
   `golang.org/x/crypto`, and `golang.org/x/sys`
   (<https://raw.githubusercontent.com/cloudflare/circl/main/go.mod>). It
   offers the instantiation with public keys on G2 and signatures on G1, named `KeyG2SigG1`; its
   built-in `dstG1` is the tag in point 2; and `Verify` hashes the message
   to the curve and checks the key's subgroup membership
   (<https://raw.githubusercontent.com/cloudflare/circl/main/sign/bls/bls.go>,
   <https://pkg.go.dev/github.com/cloudflare/circl/sign/bls>). Releases
   v1.6.4 and v1.6.5 reject the identity signature and trailing bytes
   (<https://github.com/cloudflare/circl/releases>); v1.6.5 is the version
   chosen. The README cautions that the library has not been independently
   audited; abcd's use is verification over public inputs only (a public
   key, a public round, a public signature), so the constant-time concerns
   an audit would weigh do not arise.
4. **The check, in one shape** (tier A for the primitives; the randomness
   equation is tier C). Unmarshal the pinned key into
   `bls.PublicKey[bls.KeyG2SigG1]`; set `msg` to `sha256(round as big-endian
   uint64)`; require `bls.Verify(&pk, msg[:], sig)`. The randomness is
   `sha256(signature)`, which abcd computes itself; where a reply carries a
   `randomness` field it must equal that value. The equation of randomness
   to the hashed signature was confirmed only from a third-party client's
   README during this check and is marked secondary; the published vector in
   point 5 is what the test holds it to.
5. **A published vector, positive and negative** (tier A for the vector; the
   library run is not done). Round 1000 of quicknet answers signature
   `b44679b9a59af2ec876b1a6b1ad52ea9b1615fc3982b19576350f93447cb1125e342b73a8dd2bacbe47e4b6b63ed5e39`
   and randomness
   `fe290beca10872ef2fb164d2aa4442de4566183ec51c56ff3cd603d930e54fdd`
   (<https://api.drand.sh/52db9ba70e0cc0f6eaf7803dd07447a1f5477735fd3f661792ba94600c84e971/public/1000>).
   The negative case is the same signature against round 1001, which must
   fail. The vector was not run against circl during this check. The spec
   makes the test's first run that check, and a failure stops the build and
   is reported rather than loosened.

## Not worth adopting

- **consensys/gnark-crypto.** Audited (Kudelski Security, 2022, tier B),
  Apache-2.0, but its module carries nine direct and eight indirect
  requirements, against circl's three, for one verification.
- **drand/kyber with kyber-bls12381.** Pulls in a superset of both of the
  above, and its README asks for independent review before use.
- **drand/drand/v2 client.** Sixty-six requirements for a verification abcd
  can state in three lines.
- **supranational/blst.** The fastest and the most audited, but cgo, which
  abcd's `CGO_ENABLED=0` cross-compiles cannot carry.
- **The Go standard library.** It has no BLS or pairing package, so there is
  no dependency-free route.

## Gaps

- No primary source shows circl verifying a real drand beacon. The
  compatibility claim rests on circl's tag, groups and message shape
  matching drand's source (points 2 and 3), and is proven only when
  `TestBeaconVerify` first passes.
- The indirect lines the dependency adds to `go.mod` and `go.sum` are known
  only when `go mod tidy` runs; the spec has the build report every line it
  adds.
