package selfupdate

import "io"

// Patcher defines an interface for applying binary patches to an old item to get an updated item.
type Patcher interface {
	Patch(old io.Reader, new io.Writer, patch io.Reader) error
}
