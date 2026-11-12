package main

import "testing"

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"v0.1.0-beta.2", "v0.1.0-beta.1", true},
		{"v0.1.0-beta.1", "v0.1.0-beta.2", false}, // a build ahead of the last release
		{"v0.1.0-beta.1", "v0.1.0-beta.1", false},
		{"v0.1.0-beta.10", "v0.1.0-beta.2", true}, // numbers, not text
		{"v0.1.0", "v0.1.0-beta.9", true},         // final after its pre-releases
		{"v0.1.0-beta.9", "v0.1.0", false},
		{"v0.1.1-beta.1", "v0.1.0", true},
		{"v0.10.0", "v0.9.3", true},
		{"v1.0.0", "v0.9.3", true},
		{"v0.1.0-beta.1.1", "v0.1.0-beta.1", true},
		{"nightly", "v0.1.0", false}, // not a version
		{"v0.2.0", "dev", false},
	} {
		if got := newer(c.a, c.b); got != c.want {
			t.Errorf("newer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
