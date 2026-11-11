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

// Valeurs envoyées à Abyss Tracker. La spec publique ne les liste pas : elles viennent de l'ancien code
// du site (molbal/abyss-tracker, CreateRunHelper.php) et d'un run de test accepté par le serveur.
var (
	hullValues    = []string{"Frigate", "Destroyer", "Cruiser"}                    // noms du champ HullType des runs
	weatherValues = []string{"Electrical", "Dark", "Exotic", "Firestorm", "Gamma"} // confirmées par les runs publics
)

// Les runs partent toujours en public, avec le nom du personnage.
// Toute autre valeur publie le run en anonyme (confirmé par l'équipe d'EVE Workbench).
const privacy = "PUBLIC"

// Fit est un fit du compte Abyss Tracker.
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
		return nil, fmt.Errorf("liste des fits: %w", err)
	}
	if res.Error {
		return nil, fmt.Errorf("liste des fits: %s", res.Message)
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

// telemetry suit TelemetryRequestDto ; le serveur refuse tout champ inconnu.
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
		// shortcut: les conteneurs ne sont pas observables sans lire l'écran. L'ancien code du site classe
		// le run « tous les conteneurs pillés » quand les deux nombres sont égaux : 1/0 évite cette fausse
		// déclaration. À remplacer par un choix de session si le classement obtenu ne convient pas.
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
	loot := make([]lootItem, 0, len(c)) // jamais null : le champ est obligatoire
	for name, qty := range c {
		loot = append(loot, lootItem{Name: items[name].Name, Count: qty, TypeID: items[name].TypeID})
	}
	sort.Slice(loot, func(i, j int) bool { return loot[i].Name < loot[j].Name })
	return loot
}

// errRejected : le serveur a refusé le run ; le renvoyer tel quel ne servirait à rien.
var errRejected = errors.New("run refusé par Abyss Tracker")

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
		return "", err // réseau ou 5xx : on réessaiera
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

// marshal écrit un JSON lisible : les fichiers pending/ et incomplete/ sont aussi lus par un humain.
func marshal(v any) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return b
}
