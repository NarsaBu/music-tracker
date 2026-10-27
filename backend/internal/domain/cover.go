package domain

// CoverMinSize is the smallest side a downloadable cover must have, in pixels.
const CoverMinSize = 1200

// CoverImage describes an original front cover image in the Cover Art Archive.
type CoverImage struct {
	URL           string
	Width, Height int
	ContentType   string // image/jpeg, image/png, …
	// Size in bytes; 0 when unknown.
	Size int64
}

// AtLeast reports whether both sides are at least n pixels.
func (c CoverImage) AtLeast(n int) bool { return c.Width >= n && c.Height >= n }

// Extension is the file extension for the content type, with the dot.
func (c CoverImage) Extension() string {
	switch c.ContentType {
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}
