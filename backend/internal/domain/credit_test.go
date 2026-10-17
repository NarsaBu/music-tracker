package domain

import "testing"

func credit(parts ...string) ArtistCredit {
	// parts: id, joinphrase, id, joinphrase, …
	var c ArtistCredit
	for i := 0; i < len(parts); i += 2 {
		c = append(c, CreditName{ArtistMBID: MBID(parts[i]), Name: parts[i], JoinPhrase: parts[i+1]})
	}
	return c
}

func TestRoleOf(t *testing.T) {
	tests := []struct {
		name   string
		credit ArtistCredit
		artist MBID
		want   CreditRole
	}{
		{"solo", credit("a", ""), "a", RoleMain},
		{"first of feat.", credit("a", " feat. ", "b", ""), "a", RoleMain},
		{"feat.", credit("a", " feat. ", "b", ""), "b", RoleFeatured},
		{"Feat without dot", credit("a", " Feat ", "b", ""), "b", RoleFeatured},
		{"ft.", credit("a", " ft. ", "b", ""), "b", RoleFeatured},
		{"featuring", credit("a", " featuring ", "b", ""), "b", RoleFeatured},
		{"with", credit("a", " with ", "b", ""), "b", RoleFeatured},
		{"duet with", credit("a", " duet with ", "b", ""), "b", RoleFeatured},
		{"parenthesized feat.", credit("a", " (feat. ", "b", ")"), "b", RoleFeatured},
		{"second guest after &", credit("a", " feat. ", "b", " & ", "c", ""), "c", RoleFeatured},
		{"ampersand", credit("a", " & ", "b", ""), "b", RoleCollaboration},
		{"and", credit("a", " and ", "b", ""), "b", RoleCollaboration},
		{"x", credit("a", " x ", "b", ""), "b", RoleCollaboration},
		{"comma then &", credit("a", ", ", "b", " & ", "c", ""), "c", RoleCollaboration},
		{"vs.", credit("a", " vs. ", "b", ""), "b", RoleCollaboration},
		{"slash", credit("a", " / ", "b", ""), "b", RoleCollaboration},
		{"collaborator before guest", credit("a", " & ", "b", " feat. ", "c", ""), "b", RoleCollaboration},
		{"word containing ft is not feat", credit("a", " Left ", "b", ""), "b", RoleCollaboration},
		{"absent from credit", credit("a", ""), "z", RoleFeatured},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RoleOf(tt.credit, tt.artist); got != tt.want {
				t.Errorf("RoleOf(%q, %s) = %s, want %s", tt.credit.String(), tt.artist, got, tt.want)
			}
		})
	}
}

func TestArtistCredit_String(t *testing.T) {
	c := ArtistCredit{{Name: "Jay-Z", JoinPhrase: " feat. "}, {Name: "Rihanna", JoinPhrase: " & "}, {Name: "Kanye West"}}
	if got := c.String(); got != "Jay-Z feat. Rihanna & Kanye West" {
		t.Errorf("String() = %q", got)
	}
}
