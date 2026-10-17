package feed

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
)

func items() []domain.NotificationItem {
	at := time.Date(2026, 10, 17, 6, 3, 0, 0, time.UTC)
	return []domain.NotificationItem{
		{
			ID: "n2", CreatedAt: at.Add(time.Minute), Role: domain.RoleFeatured, FoundVia: domain.Artist{Name: "Björk"},
			ReleaseGroup: domain.ReleaseGroup{
				MBID: "00000000-0000-0000-0000-000000000002", Title: "Duet <Live>", PrimaryType: domain.PrimarySingle,
				SecondaryTypes: []string{"Live"}, FirstRelease: domain.ParseReleaseDate("2026-11-07"), ArtistCredit: "Radiohead feat. Björk",
			},
		},
		{
			ID: "n1", CreatedAt: at, Role: domain.RoleMain, FoundVia: domain.Artist{Name: "Radiohead"},
			ReleaseGroup: domain.ReleaseGroup{MBID: "00000000-0000-0000-0000-000000000001", Title: "New Album", PrimaryType: domain.PrimaryAlbum, ArtistCredit: "Radiohead"},
		},
	}
}

func TestTitle(t *testing.T) {
	its := items()
	if got := Title(its[0]); got != "[feat.] Radiohead feat. Björk — Duet <Live> (Single + Live, 2026-11-07)" {
		t.Errorf("featured title = %q", got)
	}
	if got := Title(its[1]); got != "Radiohead — New Album (Album, дата неизвестна)" {
		t.Errorf("title = %q", got)
	}
}

func TestRender_Atom(t *testing.T) {
	doc, err := Render(items(), Atom, "https://tracker.example")
	if err != nil {
		t.Fatal(err)
	}
	var atom struct {
		XMLName xml.Name `xml:"feed"`
		Entries []struct {
			ID      string `xml:"id"`
			Title   string `xml:"title"`
			Content string `xml:"content"`
			Link    struct {
				Href string `xml:"href,attr"`
			} `xml:"link"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(doc, &atom); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, doc)
	}
	if len(atom.Entries) != 2 {
		t.Fatalf("entries = %d", len(atom.Entries))
	}
	e := atom.Entries[0]
	if e.ID != "urn:musicbrainz:release-group:00000000-0000-0000-0000-000000000002" ||
		e.Link.Href != "https://musicbrainz.org/release-group/00000000-0000-0000-0000-000000000002" ||
		!strings.Contains(e.Title, "Duet <Live>") {
		t.Errorf("entry = %+v", e)
	}
	if !strings.Contains(e.Content, "front-250") || !strings.Contains(e.Content, "Duet &lt;Live&gt;") {
		t.Errorf("content = %s", e.Content)
	}
}

func TestRender_RSS(t *testing.T) {
	doc, err := Render(items(), RSS, "https://tracker.example")
	if err != nil {
		t.Fatal(err)
	}
	var rss struct {
		Items []struct {
			GUID struct {
				Value       string `xml:",chardata"`
				IsPermaLink string `xml:"isPermaLink,attr"`
			} `xml:"guid"`
		} `xml:"channel>item"`
	}
	if err := xml.Unmarshal(doc, &rss); err != nil {
		t.Fatal(err)
	}
	if len(rss.Items) != 2 || rss.Items[1].GUID.Value != "urn:musicbrainz:release-group:00000000-0000-0000-0000-000000000001" || rss.Items[1].GUID.IsPermaLink != "false" {
		t.Errorf("items = %+v", rss.Items)
	}
}

func TestETagAndParse(t *testing.T) {
	its := items()
	if ETag(its, Atom) == ETag(its, RSS) || ETag(its, Atom) == ETag(its[1:], Atom) || ETag(its, Atom) != ETag(items(), Atom) {
		t.Error("etag must depend on format and content, and be stable")
	}
	if !LastModified(its).Equal(its[0].CreatedAt) || !LastModified(nil).IsZero() {
		t.Error("last modified")
	}

	tests := map[string]struct {
		token string
		f     Format
		ok    bool
	}{
		"abc.atom": {"abc", Atom, true}, "abc.rss": {"abc", RSS, true},
		".atom": {"", "", false}, "abc.xml": {"", "", false}, "abc": {"", "", false},
	}
	for in, want := range tests {
		token, f, ok := Parse(in)
		if token != want.token || f != want.f || ok != want.ok {
			t.Errorf("Parse(%q) = %q %q %t", in, token, f, ok)
		}
	}
}
