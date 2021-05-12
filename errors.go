package dropbox

import "github.com/aos-dev/go-storage/v3/services"

var (
	// ErrEntryUnexpected is the error returned when Dropbox service has returned an unexpected kind of entry.
	ErrEntryUnexpected = services.NewErrorCode("unexpected entry")
)
