package claim

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The holder id (iss-2610290809526033) is what everything this package pushes
// names its holder by: an opaque per-account id, h_ and 32 lower-case hex
// characters (128 bits from crypto/rand), minted once into the person's abcd
// home and kept. It is no hash of anything, so it says nothing of the account
// that minted it, and the underscore cannot occur in a GitHub username, so an
// id and the login an older reservation or announcement names never collide.
// Lower case keeps a status branch named by it apart from every other on a
// case-insensitive filesystem.
//
// The login is still resolved (resolveHolder), and only locally: a hold made
// before the id names it, and is the person's own when it names the login the
// resolver yields (Self.Owns); an act whose git identity is not a person's is
// still authored as the account the token names. It is never written into a
// trailer or a ref name.
const (
	holderIDLeaf      = "holder-id"
	holderIDsLeaf     = "holder-ids"
	holderIDPrefix    = "h_"
	holderIDHexLen    = 32
	maxHolderIDBytes  = 256
	maxHolderIDsBytes = 64 << 10
)

// HolderIDEnv is the environment variable that names this account's holder
// id where the home is not kept between sessions (a hosted or ephemeral
// session): set, it is read before HolderIDFile and is the id, and it is
// never written to the file. Set to anything but one holder id, every act
// that needs the id is refused, and neither the file nor a mint stands in.
const HolderIDEnv = "ABCD_HOLDER_ID"

// HolderIDFile is where the person's holder id is kept, and
// DeclaredHolderIDsFile where they declare the ids of their other machines
// and accounts as theirs too, both in the tilde form a message names them by.
var (
	HolderIDFile          = abcdhome.Display(holderIDLeaf)
	DeclaredHolderIDsFile = abcdhome.Display(holderIDsLeaf)
)

var holderIDRe = regexp.MustCompile(`^h_[0-9a-f]{32}$`)

// envHolderID is the holder id HolderIDEnv names: set is false when the
// variable is unset or empty; a value that is set and is not one holder id
// is an error, which never echoes the value.
func envHolderID() (id string, set bool, err error) {
	v, ok := os.LookupEnv(HolderIDEnv)
	if !ok || v == "" {
		return "", false, nil
	}
	if !validHolderID(v) {
		return "", true, refuse("%s is set but is not a holder id (h_ and 32 lower-case hex characters), so which holds are yours cannot be told; "+
			"it is never replaced by %s or by a new id: set it to the id `abcd reserve --holder` prints, or unset it; %s",
			HolderIDEnv, HolderIDFile, nothingSent)
	}
	return v, true, nil
}

// mintNotice is where a fresh mint is told (RegisterMintNotice); nil tells
// no one.
var mintNotice atomic.Pointer[func(string)]

// RegisterMintNotice sets where the sentence saying a fresh holder id was
// minted goes: the front door registers one that writes it to stderr, once,
// at the mint that wrote the file. nil stops it.
func RegisterMintNotice(f func(string)) {
	if f == nil {
		mintNotice.Store(nil)
		return
	}
	mintNotice.Store(&f)
}

// MintNoticeLine is the sentence a fresh mint of id is told by.
func MintNoticeLine(id string) string {
	return fmt.Sprintf("minted a new holder id, %s, and kept it in %s: holds made under an earlier id of yours (another home, or this one before it was emptied) "+
		"read as someone else's until they lapse, and the cap counts them apart; to keep one id, set %s to it in each environment, or list your earlier ids in %s",
		id, HolderIDFile, HolderIDEnv, DeclaredHolderIDsFile)
}

// validHolderID reports whether s is a holder id: h_ and 32 lower-case hex
// characters, exactly.
func validHolderID(s string) bool { return holderIDRe.MatchString(s) }

// validHolder reports whether s may name a holder or a sender in a trailer:
// a holder id, or the GitHub username every hold made before the id names.
func validHolder(s string) bool { return validHolderID(s) || validUsername(s) }

// Self is whose holds are the reader's own: the holder id this account's home
// keeps, the ids it declares as the person's too, and the GitHub login the
// holder resolver yields, which only a hold made before the id names. The
// zero Self owns nothing.
type Self struct {
	// ID is the holder id: the one HolderIDEnv names, else the kept one; ""
	// when there is neither (a read never mints it).
	ID string
	// FromEnv is true when ID is the one HolderIDEnv names.
	FromEnv bool
	// Declared is the ids DeclaredHolderIDsFile lists, in its order.
	Declared []string
	// Login is the resolved GitHub login, "" when it cannot be told.
	Login string
	// HolderIDsRefused is the class of the refusal (HolderIDsNotRegular and
	// the rest) of DeclaredHolderIDsFile when a render's read refused it,
	// so no declared id was read from it; "" when it was read or is absent.
	// An act's read refuses instead.
	HolderIDsRefused string
	// HolderIDRefused is the class of the refusal of the holder id itself
	// when a render's read refused it (HolderIDEnvNotID for a malformed
	// HolderIDEnv, else the class of HolderIDFile's), so ID is "" though
	// an id is set or kept; "" when one was read or none is set or kept.
	HolderIDRefused string
	// paired is the GitHub login DeclaredHolderIDsFile pairs with each
	// declared id that has one beside it: the account that id is the
	// person's on, which a commit naming it is held to (asker.namesPusher);
	// a commit naming a declared id with none is held to Login.
	// It is read and compared on this machine alone, and never written into
	// anything pushed or shown.
	paired map[string]string
}

// Owns reports whether holder, a reservation's holder or an announcement's
// sender, is the reader's own: one of their holder ids exactly, or, for a
// hold made before the id, their login compared as the forge compares it.
func (s Self) Owns(holder string) bool {
	switch {
	case holder == "":
		return false
	case validHolderID(holder):
		return holder == s.ID || slices.Contains(s.Declared, holder)
	}
	return s.Login != "" && sameHolder(holder, s.Login)
}

// declaredOnly reports whether holder is an id s declares as the person's
// (Declared) and is not s's own kept id: another account's or home's of
// theirs.
func (s Self) declaredOnly(holder string) bool {
	return validHolderID(holder) && holder != s.ID && slices.Contains(s.Declared, holder)
}

// isOwn reports whether holder names the reader themselves rather than one
// of their declared ids: their kept holder id, or, on a hold made before the
// id, their login.
func (s Self) isOwn(holder string) bool {
	switch {
	case holder == "":
		return false
	case validHolderID(holder):
		return holder == s.ID
	}
	return s.Login != "" && sameHolder(holder, s.Login)
}

// alike reports whether a and b, two holder names on the trailers of one
// release, name the same holder as s reads them: the same name, compared as
// the forge compares a login, or each naming s themselves, their kept id and
// their login alike, since a withdrawal of theirs recorded before the id
// names the login and every later push of theirs names the id.
func (s Self) alike(a, b string) bool {
	return sameHolder(a, b) || s.ID != "" && s.isOwn(a) && s.isOwn(b)
}

// asKept is ws with each entry naming s by their login, a withdrawal of
// theirs recorded before the holder id, named by their kept id instead, so
// no push of theirs carries the login; ws as it is when s keeps no id.
func (s Self) asKept(ws []Withdrawal) []Withdrawal {
	if s.ID == "" {
		return ws
	}
	var out []Withdrawal
	for _, w := range ws {
		if !validHolderID(w.By) && s.isOwn(w.By) {
			w.By = s.ID
		}
		out = append(out, w)
	}
	return out
}

// pairedLogin is the login DeclaredHolderIDsFile pairs with holder, an id s
// only declares; ok is false for any other holder, and for a declared id
// with no login beside it.
func (s Self) pairedLogin(holder string) (login string, ok bool) {
	if !s.declaredOnly(holder) {
		return "", false
	}
	login, ok = s.paired[holder]
	return login, ok
}

// unpaired reports whether holder is an id s only declares with no login
// beside it: a commit naming it is held to s's own login, as the same
// person's other home under one login pushes it (asker.namesPusher), so this
// copy reads an end from it, and acts on its withdrawals, only from pushes of
// s's own login.
func (s Self) unpaired(holder string) bool {
	_, ok := s.pairedLogin(holder)
	return s.declaredOnly(holder) && !ok
}

// Known reports whether anything names the reader: a holder id or a login.
func (s Self) Known() bool { return s.ID != "" || len(s.Declared) > 0 || s.Login != "" }

// Name is how a render names the reader: their kept holder id, else their
// login, else "".
func (s Self) Name() string {
	if s.ID != "" {
		return s.ID
	}
	return s.Login
}

// OwnSelf is the person's own holder ids, the kept one minted when none is
// kept yet, with login, the account the holder resolver named: what `abcd
// reserve --holder` prints. It pushes nothing; it writes only the minted id.
// A DeclaredHolderIDsFile the guarded read refuses is read as declaring no
// id, with its class in HolderIDsRefused, as a render reads it; every other
// refusal is ownSelf's.
func OwnSelf(login string, o Options) (Self, error) { return readOwnSelf(o.home(), login, true, true) }

// ReaderSelf is whose holds are the reader's own, read as a render reads
// them: the holder id HolderIDEnv names, else the kept one, never minting
// one, the declared ids, and the login resolved with no network act. A home
// inside a git working tree that is not the account's own names no one: no
// id and no login, since the login too may be read from that home's machine
// configuration, so a render names no hold as the reader's, old login holds
// included, but for the id HolderIDEnv names, which no home supplies. A file
// that refuses, or a malformed variable, is read as naming no id, so a render
// shows the holds it cannot tell as the person's as another's, and the act
// that needs them refuses naming the file.
func ReaderSelf(root string, o Options) Self { return readerSelf(root, o) }

func readerSelf(root string, o Options) Self {
	var s Self
	id, set, err := envHolderID()
	switch {
	case set && err == nil:
		s.ID, s.FromEnv = id, true
	case set:
		s.HolderIDRefused = HolderIDEnvNotID
	}
	home := o.home()
	if home != "" && gitutil.HomeInsideWorkingTree(home) != "" {
		return s
	}
	s.Login = readerHolder(root, o)
	if home == "" {
		return s
	}
	if !set {
		if id, _, err := readHolderID(home); err == nil {
			s.ID = id
		} else {
			s.HolderIDRefused = keptIDClass(err)
		}
	}
	if ids, paired, class, err := readDeclaredHolderIDs(home); err == nil {
		s.Declared, s.paired = ids, paired
	} else {
		s.HolderIDsRefused = class
	}
	return s
}

// ownSelf is the person's own ids in home, with login: the id HolderIDEnv
// names when it is set, else the kept id, minted when mint is set and none is
// kept, and the declared ones. A file either
// reads from that refuses (holderFileRefusal), or one that holds no id, is
// refused, never replaced. A home inside a git working tree that is not the
// account's own is refused as the machine layer is (layered.MachineRoots):
// a checkout that sets HOME into its tree would otherwise choose whose holds
// are the person's, and the mint would write into the tree. The id
// HolderIDEnv names needs no home: with no home, or one inside a working
// tree, it is still the id, and the declared ids, which only a home keeps,
// are none.
func ownSelf(home, login string, mint bool) (Self, error) {
	return readOwnSelf(home, login, mint, false)
}

// readOwnSelf is ownSelf; with render set, a refused DeclaredHolderIDsFile
// declares no id and names its class (HolderIDsRefused) instead of refusing.
func readOwnSelf(home, login string, mint, render bool) (Self, error) {
	s := Self{Login: login}
	id, set, err := envHolderID()
	if err != nil {
		return Self{}, err
	}
	noHome := home == ""
	tree := ""
	if !noHome {
		tree = gitutil.HomeInsideWorkingTree(home)
	}
	switch {
	case set && (noHome || tree != ""):
		s.ID, s.FromEnv = id, true
		return s, nil
	case noHome:
		return Self{}, refuse("no home directory could be named to keep your holder id in (%s); %s", HolderIDFile, nothingSent)
	case tree != "":
		return Self{}, refuse("HOME lies inside the git working tree at %s and is not this account's home, so %s is neither read nor written there "+
			"and which holds are yours cannot be told; run abcd with your own HOME; %s", tree, HolderIDFile, nothingSent)
	}
	switch {
	case set:
		s.ID, s.FromEnv = id, true
	case mint:
		s.ID, err = holderID(home)
	default:
		s.ID, _, err = readHolderID(home)
	}
	if err != nil {
		return Self{}, err
	}
	ids, paired, class, err := readDeclaredHolderIDs(home)
	switch {
	case err != nil && render && class != "":
		s.HolderIDsRefused = class
	case err != nil:
		return Self{}, err
	default:
		s.Declared, s.paired = ids, paired
	}
	return s, nil
}

// holderID is the holder id kept in home, minted on first need.
func holderID(home string) (string, error) {
	id, kept, err := readHolderID(home)
	if err != nil || kept {
		return id, err
	}
	return mintHolderID(home)
}

// readHolderID is the holder id kept in home, read only when the file is a
// regular file this account owns and no other can write
// (fsutil.ReadHomeDeclaration, the guard every home declaration is read
// through); kept is false when there is none. A file that refuses, or that
// holds anything but one id, is an error.
func readHolderID(home string) (id string, kept bool, err error) {
	rel := abcdhome.Rel(holderIDLeaf)
	return readKeptID(func() ([]byte, fsutil.DeclarationRefusal, error) {
		return fsutil.ReadHomeDeclaration(home, rel, maxHolderIDBytes)
	})
}

// readKeptID reads the kept id through read. A racing mint's exclusive create
// precedes its write by an instant, so an empty file is read again for a
// moment before it is refused.
func readKeptID(read func() ([]byte, fsutil.DeclarationRefusal, error)) (string, bool, error) {
	for attempt := 0; ; attempt++ {
		data, refusal, err := read()
		switch refusal {
		case fsutil.DeclarationOK:
		case fsutil.DeclarationAbsent:
			return "", false, nil
		default:
			return "", true, declaredRefusal{holderIDsClass(refusal, err), holderFileRefusal(HolderIDFile, refusal, err)}
		}
		id := strings.TrimSpace(string(data))
		if validHolderID(id) {
			return id, true, nil
		}
		if id != "" {
			return "", true, declaredRefusal{HolderIDsMalformed, refuse("%s does not hold a holder id (h_ and 32 lower-case hex characters), so which holds are yours cannot be told; "+
				"it is never replaced: remove it to mint a new id, and list the old one in %s if it named holds of yours; %s",
				HolderIDFile, DeclaredHolderIDsFile, nothingSent)}
		}
		if attempt >= 20 {
			// Another run's mint may have written its id since, and may
			// already have pushed under it: removing the file now would
			// mint a second id beside one already in use.
			return "", true, declaredRefusal{HolderIDEmpty, refuse("%s is empty: another abcd run is most likely minting your holder id at this moment, so which holds are yours cannot be told yet; "+
				"retry; only if it is still empty on a second try, look at the file (a run stopped mid-write leaves it empty) before doing anything with it; %s",
				HolderIDFile, nothingSent)}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// mintHolderID mints a holder id into home: the home is made private if it
// is not there (abcdhome.DirMode), each level judged as the read judges it
// (fsutil.EnsureHomeDeclarationScope), so no id is minted, or pushed under,
// in a directory another account can change; and the id written by an
// exclusive create at abcdhome.FileMode, so two first needs racing keep one
// id; the one that finds the file made reads the winner's id back, through
// the directory it holds open. The winner alone tells the mint
// (RegisterMintNotice).
func mintHolderID(home string) (string, error) {
	dir, err := fsutil.EnsureHomeDeclarationScope(home, abcdhome.Rel(), abcdhome.DirMode)
	if errors.Is(err, fsutil.ErrHomeScopeExposed) {
		return "", refuse("%s, so no holder id is minted there; %s", err.Error(), nothingSent)
	}
	if err != nil {
		return "", refuse("cannot open %s to keep your holder id in: %s; %s", abcdhome.Display(), redacted(err), nothingSent)
	}
	defer dir.Close()
	var raw [holderIDHexLen / 2]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", refuse("cannot mint your holder id: %v; %s", err, nothingSent)
	}
	minted := holderIDPrefix + hex.EncodeToString(raw[:])
	err = fsutil.CreateExclusiveIn(dir, holderIDLeaf, []byte(minted+"\n"), abcdhome.FileMode)
	if err == nil {
		if f := mintNotice.Load(); f != nil {
			(*f)(MintNoticeLine(minted))
		}
		return minted, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return "", refuse("cannot write %s: %s; %s", HolderIDFile, redacted(err), nothingSent)
	}
	rel := abcdhome.Rel(holderIDLeaf)
	id, kept, err := readKeptID(func() ([]byte, fsutil.DeclarationRefusal, error) {
		return fsutil.ReadHomeDeclarationDenyingIn(dir, home, rel, maxHolderIDBytes, 0)
	})
	if err == nil && !kept {
		// Made and gone again between the create and the read.
		err = refuse("%s was made by another run and is gone again, so your holder id cannot be told; retry; %s", HolderIDFile, nothingSent)
	}
	return id, err
}

// declaredHolderIDs is the ids home declares as the person's too
// (DeclaredHolderIDsFile), and the login paired with each that has one: one
// id per line, optionally followed by the GitHub login of the account it is
// the person's on, a # starting a comment, blank lines passed over, read
// under the kept id's guards. An absent file declares none; one that
// refuses, a line that is not one id and at most one login, and an id
// declared twice with two different pairings are each an error naming the
// line by its number alone, never its contents.
func declaredHolderIDs(home string) ([]string, map[string]string, error) {
	ids, paired, _, err := readDeclaredHolderIDs(home)
	return ids, paired, err
}

// readDeclaredHolderIDs is declaredHolderIDs with the class of its refusal
// (holderIDsClass, or HolderIDsMalformed for a line), "" when it read.
func readDeclaredHolderIDs(home string) (ids []string, paired map[string]string, class string, err error) {
	ids, paired, err = parseDeclaredHolderIDs(home)
	var refusal declaredRefusal
	if errors.As(err, &refusal) {
		return nil, nil, refusal.class, refusal.err
	}
	return ids, paired, "", err
}

// declaredRefusal is a refusal of DeclaredHolderIDsFile with its class.
type declaredRefusal struct {
	class string
	err   error
}

func (r declaredRefusal) Error() string { return r.err.Error() }

func (r declaredRefusal) Unwrap() error { return r.err }

// keptIDClass is the class a refusal of the holder id names
// (Self.HolderIDRefused): the guard's, HolderIDsMalformed for a file that
// holds no id, HolderIDEmpty for an empty one; HolderIDsUnreadable for any
// other.
func keptIDClass(err error) string {
	var refusal declaredRefusal
	if errors.As(err, &refusal) {
		return refusal.class
	}
	return HolderIDsUnreadable
}

func parseDeclaredHolderIDs(home string) ([]string, map[string]string, error) {
	data, refusal, err := fsutil.ReadHomeDeclaration(home, abcdhome.Rel(holderIDsLeaf), maxHolderIDsBytes)
	switch refusal {
	case fsutil.DeclarationOK:
	case fsutil.DeclarationAbsent:
		return nil, nil, nil
	default:
		return nil, nil, declaredRefusal{holderIDsClass(refusal, err), holderFileRefusal(DeclaredHolderIDsFile, refusal, err)}
	}
	var ids []string
	paired := map[string]string{}
	for i, l := range strings.Split(string(data), "\n") {
		l, _, _ = strings.Cut(l, "#")
		f := strings.Fields(l)
		if len(f) == 0 {
			continue
		}
		if len(f) > 2 || !validHolderID(f[0]) || len(f) == 2 && !validUsername(f[1]) {
			return nil, nil, declaredRefusal{HolderIDsMalformed, refuse("%s line %d is not a holder id (h_ and 32 lower-case hex characters), alone or followed by one GitHub login, "+
				"so which holds are yours cannot be told; %s", DeclaredHolderIDsFile, i+1, nothingSent)}
		}
		id, login := f[0], ""
		if len(f) == 2 {
			login = f[1]
		}
		if slices.Contains(ids, id) {
			if had := paired[id]; !sameHolder(had, login) && (had != "" || login != "") {
				return nil, nil, declaredRefusal{HolderIDsMalformed, refuse("%s line %d declares an id an earlier line declares with another login, or with none, "+
					"so which account it is yours on cannot be told; %s", DeclaredHolderIDsFile, i+1, nothingSent)}
			}
			continue
		}
		ids = append(ids, id)
		if login != "" {
			paired[id] = login
		}
	}
	return ids, paired, nil
}

// The classes a refused DeclaredHolderIDsFile is named by (Self's
// HolderIDsRefused, holder_ids_refused in --json): which guard refused it,
// never what it holds.
const (
	HolderIDsNotRegular       = "not_regular_file"
	HolderIDsWritableByOthers = "writable_by_others"
	HolderIDsForeignOwner     = "owned_by_another_account"
	HolderIDsBehindSymlink    = "behind_symbolic_link"
	HolderIDsFolderExposed    = "folder_exposed"
	HolderIDsTooLarge         = "too_large"
	HolderIDsUnreadable       = "unreadable"
	HolderIDsMalformed        = "malformed"
	// HolderIDEmpty is a HolderIDFile that holds nothing, and
	// HolderIDEnvNotID a HolderIDEnv set to something that is not one
	// holder id: classes of Self's HolderIDRefused alone.
	HolderIDEmpty    = "empty"
	HolderIDEnvNotID = "env_not_holder_id"
)

// holderIDsClass is the class a refusal of the guarded read names.
func holderIDsClass(refusal fsutil.DeclarationRefusal, err error) string {
	switch refusal {
	case fsutil.DeclarationNotRegular:
		return HolderIDsNotRegular
	case fsutil.DeclarationWritableByOthers:
		return HolderIDsWritableByOthers
	case fsutil.DeclarationForeignOwner:
		return HolderIDsForeignOwner
	case fsutil.DeclarationBehindSymlink:
		return HolderIDsBehindSymlink
	case fsutil.DeclarationDirectoryExposed:
		return HolderIDsFolderExposed
	}
	if errors.Is(err, fsutil.ErrTooBig) {
		return HolderIDsTooLarge
	}
	return HolderIDsUnreadable
}

// guardWhy is each class a home declaration's guarded read refuses with, in
// words.
var guardWhy = map[string]string{
	HolderIDsNotRegular:       "it is not a regular file",
	HolderIDsWritableByOthers: "another account can write it",
	HolderIDsForeignOwner:     "another account owns it",
	HolderIDsBehindSymlink:    "a folder on its path is a symbolic link",
	HolderIDsFolderExposed:    "another account can change a folder on its path",
	HolderIDsTooLarge:         "it is larger than 64 KiB",
	HolderIDsUnreadable:       "it could not be read",
}

// HolderIDsRefusedLine is the sentence a render shows when it read no
// declared id because DeclaredHolderIDsFile was refused as class: the file,
// the class in words, and what follows, never its contents or a login; ""
// for no class.
func HolderIDsRefusedLine(class string) string {
	why := guardWhy[class]
	if class == HolderIDsMalformed {
		why = "a line in it is not one holder id, alone or followed by one GitHub login, or pairs an id a second way"
	}
	if why == "" {
		return ""
	}
	return fmt.Sprintf("%s is refused (%s), so no declared id is read: holds and withdrawals under the ids it declares read here as another person's, "+
		"and every act refuses, naming it, until it is fixed", DeclaredHolderIDsFile, why)
}

// HolderIDRefusedLine is the sentence a render shows when it read no holder
// id because the id set or kept was refused as class (Self.HolderIDRefused):
// the variable or the file, the class in words, and what follows, never the
// value or the file's contents; "" for no class.
func HolderIDRefusedLine(class string) string {
	if class == "" {
		return ""
	}
	if class == HolderIDEnvNotID {
		return fmt.Sprintf("%s is set but is not a holder id, so no holder id is read: holds and withdrawals under your id read here as another person's, "+
			"and every act refuses, naming it, until it is set to the id `abcd reserve --holder` prints or unset", HolderIDEnv)
	}
	why := guardWhy[class]
	switch class {
	case HolderIDEmpty:
		why = "it is empty"
	case HolderIDsMalformed:
		why = "it does not hold one holder id"
	}
	if why == "" {
		return ""
	}
	return fmt.Sprintf("%s is refused (%s), so no holder id is read: holds and withdrawals under your id read here as another person's, "+
		"and every act refuses, naming it, until it is fixed", HolderIDFile, why)
}

// holderFileRefusal is the refusal of a holder-id file one of the home
// declaration guards refused, naming the file and the guard, never its
// contents.
func holderFileRefusal(file string, refusal fsutil.DeclarationRefusal, err error) error {
	why := "could not be read"
	switch refusal {
	case fsutil.DeclarationNotRegular:
		why = "is not a regular file"
	case fsutil.DeclarationWritableByOthers:
		why = "is not to be trusted: another account can write it"
	case fsutil.DeclarationForeignOwner:
		why = "is not to be trusted: another account owns it"
	case fsutil.DeclarationBehindSymlink:
		why = "is not to be trusted: a folder on its path is a symbolic link"
	case fsutil.DeclarationDirectoryExposed:
		why = "is not to be trusted: another account can change a folder on its path"
	}
	if err != nil && refusal == fsutil.DeclarationUnreadable {
		why += " (" + redacted(err) + ")"
	}
	return refuse("%s %s, so which holds are yours cannot be told; %s", file, why, nothingSent)
}
