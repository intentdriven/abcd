package dashboard

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/tailscale"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// SeenDevice is one device that opened the dashboard in this run, by the
// names Tailscale's lookup gave, so the person can see whom to remove from
// their tailnet (D6).
type SeenDevice struct {
	Device string    `json:"device"`
	Person string    `json:"person"`
	Login  string    `json:"login"`
	First  time.Time `json:"first_opened"`
	Last   time.Time `json:"last_opened"`
}

// seenFileShape is the devices-seen file.
type seenFileShape struct {
	Devices []SeenDevice `json:"devices"`
}

// The devices-seen bounds: how many devices one run lists, and how often one
// device's last-opened time is rewritten.
const (
	maxSeenDevices = 256
	seenRefresh    = time.Minute
)

// seenRecorder keeps the devices-seen file for the running server.
type seenRecorder struct {
	home string
	now  func() time.Time

	mu      sync.Mutex
	byNode  map[string]int
	devices []SeenDevice
}

func newSeenRecorder(home string) *seenRecorder {
	return &seenRecorder{home: home, now: time.Now, byNode: map[string]int{}}
}

// record notes that id opened a page, writing the file when a device is new
// or its last-opened time is more than seenRefresh old. A write that fails
// is dropped: the list is for the person's eyes, never a gate.
func (s *seenRecorder) record(id tailscale.Identity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC().Truncate(time.Second)
	if i, ok := s.byNode[id.Node]; ok {
		if now.Sub(s.devices[i].Last) < seenRefresh {
			return
		}
		s.devices[i].Last = now
	} else {
		if len(s.devices) >= maxSeenDevices {
			return
		}
		s.byNode[id.Node] = len(s.devices)
		s.devices = append(s.devices, SeenDevice{Device: id.Device, Person: id.Person, Login: id.Login, First: now, Last: now})
	}
	data, err := json.MarshalIndent(seenFileShape{Devices: s.devices}, "", "  ")
	if err != nil {
		return
	}
	dir, err := fsutil.EnsureHomeScope(s.home, abcdhome.Rel(stateDir), abcdhome.DirMode)
	if err != nil {
		return
	}
	defer dir.Close()
	_ = fsutil.WriteFileAtomicInRoot(dir, seenFile, append(data, '\n'), abcdhome.FileMode)
}

// readSeen reads the devices-seen file, ordered by when each first opened it.
func readSeen(home string) ([]SeenDevice, error) {
	data, refusal, err := fsutil.ReadHomeDeclaration(home, abcdhome.Rel(stateDir, seenFile), maxStateBytes)
	switch refusal {
	case fsutil.DeclarationOK:
	case fsutil.DeclarationAbsent:
		return []SeenDevice{}, nil
	default:
		return nil, fmt.Errorf("%s cannot be read safely: %v", abcdhome.Display(stateDir, seenFile), err)
	}
	var f seenFileShape
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%s is not a devices list this abcd wrote", abcdhome.Display(stateDir, seenFile))
	}
	if f.Devices == nil {
		f.Devices = []SeenDevice{}
	}
	sort.SliceStable(f.Devices, func(i, j int) bool { return f.Devices[i].First.Before(f.Devices[j].First) })
	return f.Devices, nil
}
