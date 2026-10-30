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
func OwnSelf(login string, o Options) (Self, error) { return ownSelf(o.home(), login, true) }

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
	if set && err == nil {
		s.ID, s.FromEnv = id, true
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
		}
	}
	if ids, err := declaredHolderIDs(home); err == nil {
		s.Declared = ids
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
	if s.Declared, err = declaredHolderIDs(home); err != nil {
		return Self{}, err
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
			return "", true, holderFileRefusal(HolderIDFile, refusal, err)
		}
		id := strings.TrimSpace(string(data))
		if validHolderID(id) {
			return id, true, nil
		}
		if id != "" {
			return "", true, refuse("%s does not hold a holder id (h_ and 32 lower-case hex characters), so which holds are yours cannot be told; "+
				"it is never replaced: remove it to mint a new id, and list the old one in %s if it named holds of yours; %s",
				HolderIDFile, DeclaredHolderIDsFile, nothingSent)
		}
		if attempt >= 20 {
			// Another run's mint may have written its id since, and may
			// already have pushed under it: removing the file now would
			// mint a second id beside one already in use.
			return "", true, refuse("%s is empty: another abcd run is most likely minting your holder id at this moment, so which holds are yours cannot be told yet; "+
				"retry; only if it is still empty on a second try, look at the file (a run stopped mid-write leaves it empty) before doing anything with it; %s",
				HolderIDFile, nothingSent)
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
// (DeclaredHolderIDsFile): one per line, a # starting a comment, blank lines
// passed over, read under the kept id's guards. An absent file declares none;
// one that refuses, or a line that is not one holder id, is an error naming
// it.
func declaredHolderIDs(home string) ([]string, error) {
	data, refusal, err := fsutil.ReadHomeDeclaration(home, abcdhome.Rel(holderIDsLeaf), maxHolderIDsBytes)
	switch refusal {
	case fsutil.DeclarationOK:
	case fsutil.DeclarationAbsent:
		return nil, nil
	default:
		return nil, holderFileRefusal(DeclaredHolderIDsFile, refusal, err)
	}
	var ids []string
	for i, l := range strings.Split(string(data), "\n") {
		l, _, _ = strings.Cut(l, "#")
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if !validHolderID(l) {
			return nil, refuse("%s line %d is not a holder id (h_ and 32 lower-case hex characters), so which holds are yours cannot be told; %s",
				DeclaredHolderIDsFile, i+1, nothingSent)
		}
		if !slices.Contains(ids, l) {
			ids = append(ids, l)
		}
	}
	return ids, nil
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
