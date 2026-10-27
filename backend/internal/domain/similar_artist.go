package domain

// SimilarArtist is an artist listened to together with another one (ListenBrainz).
type SimilarArtist struct {
	MBID           MBID
	Name           string
	Disambiguation string
	Type           string // Person, Group, …; empty when unknown
	// Score is relative: only the order and ratio within one list mean something.
	Score int
}
