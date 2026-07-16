package config

import "testing"

func TestProfileForVendorMatchesIDBeforeName(t *testing.T) {
	c := Config{Profiles: []VendorProfile{
		{VendorID: "vnd_1", VendorName: "Acme"},
		{VendorID: "vnd_2", VendorName: "vnd_1"}, // deliberately confusable name
	}}

	p, ok := c.ProfileForVendor("vnd_1")
	if !ok || p.VendorID != "vnd_1" {
		t.Fatalf("expected ID match to win, got %+v", p)
	}
}

func TestProfileForVendorMatchesNameCaseInsensitively(t *testing.T) {
	c := Config{Profiles: []VendorProfile{
		{VendorID: "vnd_1", VendorName: "Bethesda Game Studios"},
	}}

	p, ok := c.ProfileForVendor("bethesda game studios")
	if !ok || p.VendorID != "vnd_1" {
		t.Fatalf("expected case-insensitive name match, got %+v ok=%v", p, ok)
	}
}

func TestProfileForVendorNoMatch(t *testing.T) {
	c := Config{Profiles: []VendorProfile{{VendorID: "vnd_1", VendorName: "Acme"}}}

	_, ok := c.ProfileForVendor("nonexistent")
	if ok {
		t.Fatal("expected no match")
	}
}

func TestActiveProfile(t *testing.T) {
	c := Config{
		ActiveVendorID: "vnd_2",
		Profiles: []VendorProfile{
			{VendorID: "vnd_1", VendorName: "Acme"},
			{VendorID: "vnd_2", VendorName: "Globex"},
		},
	}

	p, ok := c.ActiveProfile()
	if !ok || p.VendorID != "vnd_2" {
		t.Fatalf("expected active profile vnd_2, got %+v ok=%v", p, ok)
	}
}

func TestActiveProfileNoneSet(t *testing.T) {
	c := Config{Profiles: []VendorProfile{{VendorID: "vnd_1"}}}

	_, ok := c.ActiveProfile()
	if ok {
		t.Fatal("expected no active profile when ActiveVendorID is empty")
	}
}

func TestUpsertProfileAppendsNew(t *testing.T) {
	var c Config
	c.UpsertProfile(VendorProfile{VendorID: "vnd_1", VendorName: "Acme", Token: "t1"})

	if len(c.Profiles) != 1 || c.Profiles[0].Token != "t1" {
		t.Fatalf("unexpected profiles: %+v", c.Profiles)
	}
}

func TestUpsertProfileReplacesExisting(t *testing.T) {
	c := Config{Profiles: []VendorProfile{
		{VendorID: "vnd_1", VendorName: "Acme", Token: "old"},
		{VendorID: "vnd_2", VendorName: "Globex", Token: "unchanged"},
	}}

	c.UpsertProfile(VendorProfile{VendorID: "vnd_1", VendorName: "Acme Renamed", Token: "new"})

	if len(c.Profiles) != 2 {
		t.Fatalf("expected replace not append, got %d profiles", len(c.Profiles))
	}
	p, _ := c.ProfileForVendor("vnd_1")
	if p.Token != "new" || p.VendorName != "Acme Renamed" {
		t.Fatalf("expected profile to be replaced, got %+v", p)
	}
	other, _ := c.ProfileForVendor("vnd_2")
	if other.Token != "unchanged" {
		t.Fatalf("expected other profile untouched, got %+v", other)
	}
}

func TestRemoveProfile(t *testing.T) {
	c := Config{Profiles: []VendorProfile{
		{VendorID: "vnd_1"},
		{VendorID: "vnd_2"},
	}}

	c.RemoveProfile("vnd_1")

	if len(c.Profiles) != 1 || c.Profiles[0].VendorID != "vnd_2" {
		t.Fatalf("unexpected profiles after removal: %+v", c.Profiles)
	}
}

func TestRemoveProfileNoMatchIsNoop(t *testing.T) {
	c := Config{Profiles: []VendorProfile{{VendorID: "vnd_1"}}}

	c.RemoveProfile("nonexistent")

	if len(c.Profiles) != 1 {
		t.Fatalf("expected no change, got %+v", c.Profiles)
	}
}

func TestMergeDefaultsCarriesProfilesFromFile(t *testing.T) {
	flagsOnly := Config{APIBaseURL: "https://api.forgebit.io"}
	file := Config{
		APIBaseURL:     "https://api.forgebit.test",
		ActiveVendorID: "vnd_1",
		Profiles:       []VendorProfile{{VendorID: "vnd_1", Token: "t1"}},
	}

	flagsOnly.MergeDefaults(file)

	if flagsOnly.APIBaseURL != "https://api.forgebit.io" {
		t.Fatalf("expected flag-set APIBaseURL to win, got %q", flagsOnly.APIBaseURL)
	}
	if flagsOnly.ActiveVendorID != "vnd_1" || len(flagsOnly.Profiles) != 1 {
		t.Fatalf("expected profiles/active vendor to come from file: %+v", flagsOnly)
	}
}

func TestMergeDefaultsFallsBackToDefaultAPIBaseURL(t *testing.T) {
	var c Config
	c.MergeDefaults(Config{})

	if c.APIBaseURL != DefaultAPIBaseURL {
		t.Fatalf("expected default API base URL, got %q", c.APIBaseURL)
	}
}
