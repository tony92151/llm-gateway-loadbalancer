package monitor

import _ "embed"

//go:embed static/index.html
var indexHTML []byte

func IndexHTML() []byte {
	out := make([]byte, len(indexHTML))
	copy(out, indexHTML)
	return out
}
