// Package modules contains concrete Phase implementations (apt, uv_tool, etc).
// Each module file registers itself via Register in its init() function.
//
// Import this package blank to activate all registered module factories:
//
//	import _ "github.com/justanotherspy/sproot/internal/phase/modules"
package modules
