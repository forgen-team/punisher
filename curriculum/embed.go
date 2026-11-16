// Package curriculum embeds the versioned curriculum seed (*.yaml).
// Each file describes one knowledge area; see README.md for the schema.
package curriculum

import "embed"

//go:embed *.yaml
var Files embed.FS

// Glossary is the term dictionary (glossary/*.yaml, internal/glossary).
//
//go:embed glossary
var Glossary embed.FS
