package domain

import "time"

// Review is a user review of a release group (CritiqueBrainz).
type Review struct {
	ID     string
	Author string
	// Rating is 1–5; 0 when the review has no rating.
	Rating      int
	Text        string
	Language    string
	PublishedAt time.Time
	// License names the license the text is published under (Creative Commons).
	License string
}

// Reviews is one page of reviews with totals over all of them.
type Reviews struct {
	Items []Review
	Total int
	// Average is the mean rating of the rated reviews; zero Rated means none.
	Average float64
	Rated   int
}
