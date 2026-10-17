package domain

// PrimaryType is a MusicBrainz release group primary type, stored lowercase.
type PrimaryType string

const (
	PrimaryAlbum     PrimaryType = "album"
	PrimarySingle    PrimaryType = "single"
	PrimaryEP        PrimaryType = "ep"
	PrimaryBroadcast PrimaryType = "broadcast"
	PrimaryOther     PrimaryType = "other"
)

// PrimaryTypes lists every known primary type.
var PrimaryTypes = []PrimaryType{PrimaryAlbum, PrimarySingle, PrimaryEP, PrimaryBroadcast, PrimaryOther}

// SecondaryTypes lists MusicBrainz secondary types, spelled exactly as in the API.
var SecondaryTypes = []string{
	"Compilation", "Soundtrack", "Spokenword", "Interview", "Audiobook", "Audio drama",
	"Live", "Remix", "DJ-mix", "Mixtape/Street", "Demo", "Field recording",
}
