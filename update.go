package main

import (
	"cmp"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// version is the name of the GitHub release this build belongs to. Change it before each release:
// the release tag must be the same.
var version = "v0.1.0-beta.4"

// /releases/latest skips pre-releases, so the newest release is taken from the list instead.
const releasesAPI = "https://api.github.com/repos/Ipicky22/EVE-Online-AbyssTracker-Autosubmit/releases?per_page=1"

const updateEvery = 6 * time.Hour // abysstracker-autosubmit can stay open for days

type release struct {
	Tag string `json:"tag_name"`
	URL string `json:"html_url"` // the release page, where abysstracker-autosubmit.exe is attached
}

func latestRelease() (release, error) {
	req, _ := http.NewRequest("GET", releasesAPI, nil)
	var list []release
	if err := doJSON(req, &list); err != nil || len(list) == 0 {
		return release{}, err
	}
	return list[0], nil
}

// watchUpdates looks for a newer release at startup and then from time to time, for the page to show it.
func (a *App) watchUpdates() {
	for {
		if r, err := latestRelease(); err != nil {
			log.Printf("update check: %v", err)
		} else if newer(r.Tag, version) {
			a.mu.Lock()
			a.update = r
			a.mu.Unlock()
		}
		time.Sleep(updateEvery)
	}
}

var versionRE = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`)

// newer reports whether version a comes after version b. Versions look like v1.2.3 or v1.2.3-beta.4,
// and a final version comes after its pre-releases. A name that is not a version is never newer.
func newer(a, b string) bool {
	if !versionRE.MatchString(a) || !versionRE.MatchString(b) {
		return false
	}
	coreA, preA, _ := strings.Cut(a[1:], "-")
	coreB, preB, _ := strings.Cut(b[1:], "-")
	if c := cmpDotted(coreA, coreB); c != 0 {
		return c > 0
	}
	if preA == "" || preB == "" {
		return preA == "" && preB != ""
	}
	return cmpDotted(preA, preB) > 0
}

// cmpDotted compares dot-separated parts one by one, as numbers when both are numbers.
func cmpDotted(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := range min(len(as), len(bs)) {
		c := strings.Compare(as[i], bs[i])
		x, errX := strconv.Atoi(as[i])
		y, errY := strconv.Atoi(bs[i])
		if errX == nil && errY == nil {
			c = cmp.Compare(x, y)
		}
		if c != 0 {
			return c
		}
	}
	return cmp.Compare(len(as), len(bs))
}
