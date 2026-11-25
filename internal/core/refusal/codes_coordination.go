package refusal

// codes_coordination.go — the codes of the verbs that coordinate work between
// sessions, people and machines: reserve, announce, peers, status, embargo and
// banlist (spc-2610301431108175 step 3). Their flag-combination and operand
// refusals carry the usage codes; these are the refusals of a call that parsed.

const (
	// reserve's own refusals.
	ReserveOutsideCheckout   Code = "reserve/outside-checkout"
	ReserveAlreadyHeld       Code = "reserve/already-held"
	ReserveExceededCap       Code = "reserve/exceeded-cap"
	ReserveNotHeld           Code = "reserve/not-held"
	ReserveUnreachableOrigin Code = "reserve/unreachable-origin"
	ReserveUnknownHolder     Code = "reserve/unknown-holder"
	ReserveInvalidRequest    Code = "reserve/invalid-request"

	// announce's own refusals.
	AnnounceOutsideCheckout   Code = "announce/outside-checkout"
	AnnounceInvalidRequest    Code = "announce/invalid-request"
	AnnounceNotPublic         Code = "announce/not-public"
	AnnounceUnreachableOrigin Code = "announce/unreachable-origin"

	// peers' and status's own refusals.
	PeersOutsideCheckout  Code = "peers/outside-checkout"
	StatusOutsideCheckout Code = "status/outside-checkout"

	// embargo's own refusals.
	EmbargoOutsideCheckout Code = "embargo/outside-checkout"
	EmbargoUnknownHome     Code = "embargo/unknown-home"
	EmbargoNotAway         Code = "embargo/not-away"
	EmbargoInvalidRequest  Code = "embargo/invalid-request"

	// banlist's own refusals.
	BanlistInvalidEntry   Code = "banlist/invalid-entry"
	BanlistDuplicateKey   Code = "banlist/duplicate-key"
	BanlistUnknownKey     Code = "banlist/unknown-key"
	BanlistUnmanagedEntry Code = "banlist/unmanaged-entry"
	BanlistLegacyStore    Code = "banlist/legacy-store"
	BanlistInvalidStore   Code = "banlist/invalid-store"
)

// outsideCheckout is the explanation every <verb>/outside-checkout code
// shares: the verb reads or writes one repository's records, and the call was
// made where git names no repository.
const outsideCheckout = "The working directory is not inside a git checkout, and the verb reads or writes\n" +
	"one repository's records. Run it from the repository's checkout; nothing was\n" +
	"read or written."

func init() {
	register(
		Entry{
			Code:        ReserveOutsideCheckout,
			Summary:     "a reservation is per-repository, and this call was made outside a checkout",
			Explanation: outsideCheckout,
			Next:        NoNext,
			RaisedBy:    []string{"abcd reserve"},
		},
		Entry{
			Code:    ReserveAlreadyHeld,
			Summary: "someone else holds the record, or a file of its footprint, so back off",
			Explanation: "Another person, or your own other copy, holds a live reservation of the record or\n" +
				"of a file this footprint names, or another release is being cut. Nothing was\n" +
				"pushed. Work on something else, or wait for the hold to end; `abcd peers` lists\n" +
				"every reservation and when it lapses. Your own other copy is moved here with\n" +
				"--take-over.",
			Next:     "abcd peers",
			RaisedBy: []string{"abcd reserve"},
		},
		Entry{
			Code:    ReserveExceededCap,
			Summary: "you already hold as many live reservations as this machine allows",
			Explanation: "A person holds at most reservation.max_live live reservations at once (the\n" +
				"machine's ~/.abcd.noindex/config.json). The refusal lists the ones you hold.\n" +
				"Release one you have finished with, then reserve again; nothing was pushed.",
			Next:     "abcd reserve <record-id> --release",
			RaisedBy: []string{"abcd reserve"},
		},
		Entry{
			Code:    ReserveNotHeld,
			Summary: "the act needs your own reservation from this copy, and the record has none",
			Explanation: "A renewal or a release acts on your own reservation, made from this copy. The\n" +
				"record's reservation is absent, released, another person's or another copy's,\n" +
				"or was renewed since; the refusal says which. Nothing was pushed.",
			Next:     "abcd peers",
			RaisedBy: []string{"abcd reserve"},
		},
		Entry{
			Code:    ReserveUnreachableOrigin,
			Summary: "origin could not be read or written, so the reservations cannot be judged",
			Explanation: "Reservations live on the remote named origin. The fetch of the reservations, or\n" +
				"the push of this act, failed; the refusal quotes git's reason. Check that origin\n" +
				"is reachable and that you may push to it, then run the act again.",
			Next:     NoNext,
			RaisedBy: []string{"abcd reserve"},
		},
		Entry{
			Code:    ReserveUnknownHolder,
			Summary: "whose account a reservation would be made as cannot be told",
			Explanation: "A reservation names the person who holds it and the copy it was made from.\n" +
				"Neither git's identity nor the forge named the person (git's email is not a\n" +
				"forge no-reply address and the forge could not be asked), or the copy's holder\n" +
				"id cannot be read or kept under this HOME: the refusal says which. Set git's\n" +
				"user.email to your no-reply address, sign the forge's command-line tool in, or\n" +
				"run from your own account's HOME. Nothing was written.",
			Next:     NoNext,
			RaisedBy: []string{"abcd reserve"},
		},
		Entry{
			Code:    ReserveInvalidRequest,
			Summary: "the reservation was refused on what it asks for; the refusal names why",
			Explanation: "The record id is not a reservable one, a footprint path or a lease is outside\n" +
				"its shape, the checkout has no origin or no local tier, the record is under\n" +
				"embargo, or the act does not apply to the record: the refusal names which and\n" +
				"what to do. Nothing was pushed.",
			Next:     "abcd explain reserve",
			RaisedBy: []string{"abcd reserve"},
		},
		Entry{
			Code:        AnnounceOutsideCheckout,
			Summary:     "an announcement is per-repository, and this call was made outside a checkout",
			Explanation: outsideCheckout,
			Next:        NoNext,
			RaisedBy:    []string{"abcd announce"},
		},
		Entry{
			Code:    AnnounceInvalidRequest,
			Summary: "the announcement was refused on what it says; the refusal names why",
			Explanation: "The kind is not one of the fixed kinds, the item is not an issue, an intent or a\n" +
				"pull request, the expiry is outside its range, the checkout has no origin, or\n" +
				"the item is under embargo: the refusal names which. Nothing was pushed.",
			Next:     "abcd explain announce",
			RaisedBy: []string{"abcd announce"},
		},
		Entry{
			Code:    AnnounceNotPublic,
			Summary: "an announcement names only a record already on origin's default branch",
			Explanation: "Everyone reads an announcement, so it may name only a record that is public: one\n" +
				"on origin's default branch as just fetched. Land the record first, then announce\n" +
				"it. Nothing was pushed.",
			Next:     NoNext,
			RaisedBy: []string{"abcd announce"},
		},
		Entry{
			Code:    AnnounceUnreachableOrigin,
			Summary: "origin could not be read or did not take the announcement",
			Explanation: "Announcements live on the remote named origin. Reading its default branch, or\n" +
				"pushing the announcement, failed; the refusal quotes git's reason and says\n" +
				"whether a refused push may still have reached origin. Fetch origin and look at\n" +
				"your status branch before announcing again.",
			Next:     "abcd status",
			RaisedBy: []string{"abcd announce"},
		},
		Entry{
			Code:        PeersOutsideCheckout,
			Summary:     "the peer listing is per-repository, and this call was made outside a checkout",
			Explanation: outsideCheckout,
			Next:        NoNext,
			RaisedBy:    []string{"abcd peers"},
		},
		Entry{
			Code:        StatusOutsideCheckout,
			Summary:     "the status listing is per-repository, and this call was made outside a checkout",
			Explanation: outsideCheckout,
			Next:        NoNext,
			RaisedBy:    []string{"abcd status"},
		},
		Entry{
			Code:        EmbargoOutsideCheckout,
			Summary:     "the embargo store is kept per repository, and this call was made outside a checkout",
			Explanation: outsideCheckout,
			Next:        NoNext,
			RaisedBy:    []string{"abcd embargo intake"},
		},
		Entry{
			Code:    EmbargoUnknownHome,
			Summary: "the home directory cannot be named, so there is no store to keep the embargo in",
			Explanation: "The embargo store lives under your abcd home, which is found from HOME, and HOME\n" +
				"is unset or empty. Run the verb from your own account's environment; nothing\n" +
				"was written.",
			Next:     NoNext,
			RaisedBy: []string{"abcd embargo intake"},
		},
		Entry{
			Code:    EmbargoNotAway,
			Summary: "the person is not away, so there is no return to queue a step for",
			Explanation: "Queueing waits for the person's return, and they are not away, came back while\n" +
				"the steps were queued, or whether they are away cannot be read: the refusal\n" +
				"says which and names any step queued before it. Tell them instead.",
			Next:     NoNext,
			RaisedBy: []string{"abcd embargo queue"},
		},
		Entry{
			Code:    EmbargoInvalidRequest,
			Summary: "the embargo store cannot act on the request; the refusal names the remedy",
			Explanation: "There is no store for this repository yet, the advisory has no verdict or no\n" +
				"reason, a release tag is outside its shape, or there is nothing to check: the\n" +
				"refusal names which and the command that gets past it. Nothing was written.",
			Next: "abcd embargo intake",
			RaisedBy: []string{"abcd embargo check", "abcd embargo close", "abcd embargo publish",
				"abcd embargo triage"},
		},
		Entry{
			Code:    BanlistInvalidEntry,
			Summary: "the key, pattern or severity given is not one the banlist takes",
			Explanation: "A key is a short slug, a pattern a regular expression that compiles, and a\n" +
				"severity one of the store's levels; the refusal names which part is wrong,\n" +
				"never the pattern itself. Nothing was written.",
			Next:     "abcd explain banlist add",
			RaisedBy: []string{"abcd banlist add"},
		},
		Entry{
			Code:    BanlistDuplicateKey,
			Summary: "the layer already holds an entry under that key",
			Explanation: "Each key names one entry of its layer. Remove the entry first, or choose another\n" +
				"key; nothing was written.",
			Next:     "abcd banlist list",
			RaisedBy: []string{"abcd banlist add"},
		},
		Entry{
			Code:    BanlistUnknownKey,
			Summary: "the layer holds no entry under that key",
			Explanation: "No entry of the layer named has that key. List the layer to see its keys;\n" +
				"nothing was written.",
			Next:     "abcd banlist list",
			RaisedBy: []string{"abcd banlist remove"},
		},
		Entry{
			Code:    BanlistUnmanagedEntry,
			Summary: "the entry was written by hand, so the verb leaves it alone",
			Explanation: "The public layer holds entries curated by hand in the docs-lint configuration as\n" +
				"well as those the verb adds, and the verb changes only its own. Edit the\n" +
				"configuration in a reviewed change instead; nothing was written.",
			Next:     NoNext,
			RaisedBy: []string{"abcd banlist remove"},
		},
		Entry{
			Code:    BanlistLegacyStore,
			Summary: "the private store predates the keyed format and must be migrated first",
			Explanation: "The private store's first line does not declare the keyed format, so its lines\n" +
				"are whole-line patterns with no keys. `abcd banlist migrate` keys them in place;\n" +
				"then run the act again. Nothing was written.",
			Next:     "abcd banlist migrate",
			RaisedBy: []string{"abcd banlist add"},
		},
		Entry{
			Code:    BanlistInvalidStore,
			Summary: "a banlist store is absent, malformed, locked or not ignored by git",
			Explanation: "The layer's store could not be used as it stands: it is absent, does not parse,\n" +
				"is held by another run, or (the private one) is not ignored by git, so a pattern\n" +
				"could be committed. The refusal names which; nothing was written.",
			Next:     NoNext,
			RaisedBy: []string{"abcd banlist remove"},
		},
	)
}
