// Package migrations embeds SQL migrations into the binary.
// Naming: NNNNNN_description.up.sql / NNNNNN_description.down.sql.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
