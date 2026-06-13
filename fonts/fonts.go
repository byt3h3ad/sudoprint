// Package fonts embeds the bundled TTF assets.
package fonts

import _ "embed"

//go:embed JetBrainsMono-Regular.ttf
var Regular []byte
