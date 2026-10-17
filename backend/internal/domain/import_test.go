package domain

import "testing"

func cand(mbid string, score int) ArtistCandidate {
	return ArtistCandidate{Artist: Artist{MBID: MBID(mbid)}, Score: score}
}

func TestAutoMatch(t *testing.T) {
	tests := []struct {
		name  string
		cands []ArtistCandidate
		want  MBID // "" = needs review
	}{
		{"nothing found", nil, ""},
		{"single confident", []ArtistCandidate{cand("a", 100)}, "a"},
		{"single at threshold", []ArtistCandidate{cand("a", 95)}, "a"},
		{"single below threshold", []ArtistCandidate{cand("a", 94)}, ""},
		{"clear lead", []ArtistCandidate{cand("a", 100), cand("b", 90)}, "a"},
		{"namesakes", []ArtistCandidate{cand("a", 100), cand("b", 100)}, ""},
		{"lead too small", []ArtistCandidate{cand("a", 100), cand("b", 91), cand("c", 50)}, ""},
		{"unsorted input", []ArtistCandidate{cand("c", 40), cand("b", 80), cand("a", 98)}, "a"},
		{"runner-up after best", []ArtistCandidate{cand("a", 99), cand("c", 40), cand("b", 95)}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := AutoMatch(tt.cands)
			if ok != (tt.want != "") || got.MBID != tt.want {
				t.Errorf("AutoMatch = %q, %t; want %q", got.MBID, ok, tt.want)
			}
		})
	}
}

func TestParseArtistRef(t *testing.T) {
	const mbid = "5b11f4ce-a62d-471e-81fc-a69a8278c7da"
	tests := []struct {
		in   string
		want MBID
	}{
		{mbid, mbid},
		{"https://musicbrainz.org/artist/" + mbid, mbid},
		{"https://MusicBrainz.org/artist/5B11F4CE-A62D-471E-81FC-A69A8278C7DA/releases", mbid},
		{"musicbrainz.org/artist/" + mbid + "?tab=1", mbid},
		{"https://musicbrainz.org/release-group/" + mbid, ""},
		{"https://musicbrainz.org/artist/" + mbid + "0", ""},
		{"Nirvana", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got, ok := ParseArtistRef(tt.in)
		if ok != (tt.want != "") || got != tt.want {
			t.Errorf("ParseArtistRef(%q) = %q, %t; want %q", tt.in, got, ok, tt.want)
		}
	}
}
