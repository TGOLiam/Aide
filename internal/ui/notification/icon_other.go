//go:build !darwin

package notification

import (
	_ "embed"
)

//go:embed aide-icon-solo.png
var Icon []byte
