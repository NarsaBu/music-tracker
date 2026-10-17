package domain

import (
	"regexp"
	"strings"
)

// CreditName is one element of a MusicBrainz artist credit. JoinPhrase is the text
// after the element: "Jay-Z" + " feat. " + "Rihanna".
type CreditName struct {
	ArtistMBID MBID
	Name       string // as credited, may differ from the artist name
	JoinPhrase string
}

type ArtistCredit []CreditName

// String renders the credit as printed on the cover, e.g. "A feat. B & C".
func (c ArtistCredit) String() string {
	var b strings.Builder
	for _, n := range c {
		b.WriteString(n.Name)
		b.WriteString(n.JoinPhrase)
	}
	return b.String()
}

// CreditRole is the role of an artist in a release group credit (ADR-9).
type CreditRole string

const (
	RoleMain          CreditRole = "main"
	RoleCollaboration CreditRole = "collaboration"
	RoleFeatured      CreditRole = "featured"
)

// CreditRoles lists every role.
var CreditRoles = []CreditRole{RoleMain, RoleCollaboration, RoleFeatured}

// featuredPhrase matches join phrases that introduce guest artists. MusicBrainz
// editors spell them in many ways, so the list is deliberately broad.
var featuredPhrase = regexp.MustCompile(`(?i)(^|[^\pL])(feat\.?|ft\.?|featuring|with|duet with|w/)([^\pL]|$)`)

// RoleOf classifies an artist in a credit (docs/07-musicbrainz.md §5):
// first credited — main; after a "feat."-like phrase (anywhere earlier) — featured;
// otherwise ("&", "and", "x", ",", "/", "vs.") — collaboration.
// An artist missing from the credit (alias, merge) is featured: never notify by mistake.
func RoleOf(credit ArtistCredit, artist MBID) CreditRole {
	idx := -1
	for i, n := range credit {
		if n.ArtistMBID == artist {
			idx = i
			break
		}
	}
	switch idx {
	case -1:
		return RoleFeatured
	case 0:
		return RoleMain
	}
	for _, n := range credit[:idx] {
		if featuredPhrase.MatchString(n.JoinPhrase) {
			return RoleFeatured
		}
	}
	return RoleCollaboration
}
