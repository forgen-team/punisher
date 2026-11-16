package curriculum

import "embed"

// Samples are ready-made lectures shipped with the app (samples/*.json) so a
// new user can read one before generating anything (design v3 §9 O3).
//
//go:embed samples
var Samples embed.FS
