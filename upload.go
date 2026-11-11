package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
)

const (
	ewbBase = "https://api.eveworkbench.com/v1"
	runURL  = "https://abysstracker.com/run/"
)

// Values sent to Abyss Tracker. The public spec does not list them: they come from the site's former code
// (molbal/abyss-tracker, CreateRunHelper.php) and from a test run accepted by the server.
var (
	hullValues    = []string{"Frigate", "Destroyer", "Cruiser"}                    // names of the runs' HullType field
	weatherValues = []string{"Electrical", "Dark", "Exotic", "Firestorm", "Gamma"} // confirmed by public runs
)

// Runs are always sent as public, with the character's name.
// Any other value publishes the run anonymously (confirmed by the EVE Workbench team).
const privacy = "PUBLIC"

// Fit is a fit of the Abyss Tracker account.
type Fit struct {
	ID   string
	Name string
	Ship struct {
		Name   string
		TypeID int
	}
}

func fetchFits(apiKey string) ([]Fit, error) {
	req, _ := http.NewRequest("GET", ewbBase+"/fits/list?onlyAbyssTracker=true", nil)
	req.Header.Set("X-API-KEY", apiKey)
	var res struct {
		Error   bool
		Message string
		Fits    []Fit
	}
	if err := doJSON(req, &res); err != nil {
		return nil, fmt.Errorf("fits list: %w", err)
	}
	if res.Error {
		return nil, fmt.Errorf("fits list: %s", res.Message)
	}
	return res.Fits, nil
}

type lootItem struct {
	Name   string `json:"name"`
	Count  int    `json:"count"`
	TypeID int    `json:"typeId"`
}

type telemetryShip struct {
	ForeignID     string `json:"foreignId"`
	CharacterName string `json:"characterName,omitempty"`
	FittingName   string `json:"fittingName,omitempty"`
	ShipName      string `json:"shipName,omitempty"`
	ShipTypeID    int    `json:"shipTypeId,omitempty"`
}

// telemetry follows TelemetryRequestDto; the server rejects any unknown field.
type telemetry struct {
	RecordUUID         string          `json:"recordUuid"`
	Privacy            string          `json:"privacy"`
	ShipType           string          `json:"shipType"`
	RunStartTime       int64           `json:"runStartTime"`
	RunDurationSeconds int             `json:"runDurationSeconds"`
	Weather            string          `json:"weather"`
	Tier               int             `json:"tier"`
	CharacterID        int             `json:"characterId"`
	CharacterName      string          `json:"characterName"`
	Success            bool            `json:"success"`
	CansSpawned        int             `json:"cansSpawned"`
	CansLooted         int             `json:"cansLooted"`
	LootGained         []lootItem      `json:"lootGained"`
	LootLost           []lootItem      `json:"lootLost"`
	Ships              []telemetryShip `json:"ships,omitempty"`
}

func buildTelemetry(run *Run, s Session, ch Character, items map[string]Item) telemetry {
	t := telemetry{
		RecordUUID:         newUUID(),
		Privacy:            privacy,
		ShipType:           s.Hull,
		RunStartTime:       run.Start.Unix(),
		RunDurationSeconds: int(run.Duration.Seconds()),
		Weather:            s.Weather,
		Tier:               s.Tier,
		CharacterID:        ch.ID,
		CharacterName:      ch.Name,
		Success:            run.Success,
		// shortcut: cans cannot be observed without reading the screen. The site's former code classifies
		// the run as "all cans looted" when both numbers are equal: 1/0 avoids that false
		// statement. Replace with a session setting if the resulting classification is not suitable.
		CansSpawned: 1,
		CansLooted:  0,
		LootGained:  toLoot(run.Gained, items),
		LootLost:    toLoot(run.Lost, items),
	}
	if s.FitID != "" {
		t.Ships = []telemetryShip{{ForeignID: s.FitID, CharacterName: ch.Name,
			FittingName: s.FitName, ShipName: s.ShipName, ShipTypeID: s.ShipTypeID}}
	}
	return t
}

func toLoot(c Cargo, items map[string]Item) []lootItem {
	loot := make([]lootItem, 0, len(c)) // never null: the field is required
	for name, qty := range c {
		loot = append(loot, lootItem{Name: items[name].Name, Count: qty, TypeID: items[name].TypeID})
	}
	sort.Slice(loot, func(i, j int) bool { return loot[i].Name < loot[j].Name })
	return loot
}

// errRejected: the server rejected the run; resending it unchanged would be pointless.
var errRejected = errors.New("run rejected by Abyss Tracker")

func sendTelemetry(apiKey string, body []byte) (runID string, err error) {
	req, _ := http.NewRequest("POST", ewbBase+"/abyss-tracker/telemetry", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", apiKey)
	var res struct {
		Error   bool
		Message string
		RunID   string
	}
	if err := doJSON(req, &res); err != nil {
		var he *httpError
		if errors.As(err, &he) && he.Status >= 400 && he.Status < 500 && he.Status != 429 {
			return "", fmt.Errorf("%w: %v", errRejected, err)
		}
		return "", err // network or 5xx: will be retried
	}
	if res.Error {
		return "", fmt.Errorf("%w: %s", errRejected, res.Message)
	}
	return res.RunID, nil
}

func newUUID() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// marshal writes readable JSON: the pending/ and incomplete/ files are also read by a human.
func marshal(v any) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return b
}
