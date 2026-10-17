package email

import (
	"embed"
	htmltemplate "html/template"
	"slices"
	"strconv"
	"strings"
	texttemplate "text/template"

	"music-tracker/backend/internal/domain"
)

//go:embed templates
var templatesFS embed.FS

var (
	textTemplates = texttemplate.Must(texttemplate.ParseFS(templatesFS, "templates/*.txt.tmpl"))
	htmlTemplates = htmltemplate.Must(htmltemplate.ParseFS(templatesFS, "templates/*.html.tmpl"))
)

type digestData struct {
	Subject  string
	Weekly   bool
	AppURL   string
	Total    int
	Sections []digestSection
}

type digestSection struct {
	Title   string
	Artists []artistGroup
}

type artistGroup struct {
	Name     string
	Releases []digestRelease
}

type digestRelease struct {
	Title    string
	Credit   string
	Type     string
	Date     string
	URL      string
	CoverURL string
}

var primaryLabels = map[domain.PrimaryType]string{
	domain.PrimaryAlbum: "Album", domain.PrimarySingle: "Single", domain.PrimaryEP: "EP",
	domain.PrimaryBroadcast: "Broadcast", domain.PrimaryOther: "Other",
}

func newDigestData(d domain.Digest, appURL string) digestData {
	data := digestData{Weekly: d.Frequency == domain.DigestWeekly, AppURL: appURL, Total: d.Len()}
	for _, s := range []struct {
		title string
		items []domain.NotificationItem
	}{
		{"Вышло", d.Released},
		{"Анонсы", d.Announcements},
		{"Гостевое участие", d.Featured},
	} {
		if len(s.items) > 0 {
			data.Sections = append(data.Sections, digestSection{Title: s.title, Artists: groupByArtist(s.items)})
		}
	}
	data.Subject = subject(data)
	return data
}

// groupByArtist groups items by the subscribed artist (alphabetically) and sorts
// each group by release date, undated last.
func groupByArtist(items []domain.NotificationItem) []artistGroup {
	byName := map[string][]domain.NotificationItem{}
	for _, it := range items {
		byName[it.FoundVia.Name] = append(byName[it.FoundVia.Name], it)
	}
	names := make([]string, 0, len(byName))
	for n := range byName {
		names = append(names, n)
	}
	slices.SortFunc(names, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) })

	groups := make([]artistGroup, 0, len(names))
	for _, name := range names {
		its := byName[name]
		slices.SortStableFunc(its, func(a, b domain.NotificationItem) int {
			da, db := a.ReleaseGroup.FirstRelease.Day, b.ReleaseGroup.FirstRelease.Day
			switch {
			case da == nil && db == nil:
				return 0
			case da == nil:
				return 1
			case db == nil:
				return -1
			}
			return da.Compare(*db)
		})
		g := artistGroup{Name: name}
		for _, it := range its {
			g.Releases = append(g.Releases, toDigestRelease(it.ReleaseGroup))
		}
		groups = append(groups, g)
	}
	return groups
}

func toDigestRelease(rg domain.ReleaseGroup) digestRelease {
	return digestRelease{
		Title:    rg.Title,
		Credit:   rg.ArtistCredit,
		Type:     typeLabel(rg),
		Date:     dateLabel(rg.FirstRelease),
		URL:      rg.MusicBrainzURL(),
		CoverURL: rg.CoverURL(),
	}
}

// typeLabel is "Album", "Album + Live", or "" when MusicBrainz has no type.
func typeLabel(rg domain.ReleaseGroup) string {
	parts := make([]string, 0, 1+len(rg.SecondaryTypes))
	if l, ok := primaryLabels[rg.PrimaryType]; ok {
		parts = append(parts, l)
	}
	parts = append(parts, rg.SecondaryTypes...)
	return strings.Join(parts, " + ")
}

func dateLabel(d domain.ReleaseDate) string {
	if d.Raw == "" {
		return "дата неизвестна"
	}
	return d.Raw
}

// subject: "Новые релизы: Radiohead, Björk и ещё 3" (weekly: "… за неделю: …").
func subject(d digestData) string {
	var names []string
	for _, s := range d.Sections {
		for _, a := range s.Artists {
			if !slices.Contains(names, a.Name) {
				names = append(names, a.Name)
			}
		}
	}
	prefix := "Новые релизы: "
	if d.Weekly {
		prefix = "Новые релизы за неделю: "
	}
	if len(names) <= 2 {
		return prefix + strings.Join(names, ", ")
	}
	return prefix + strings.Join(names[:2], ", ") + " и ещё " + strconv.Itoa(len(names)-2)
}
