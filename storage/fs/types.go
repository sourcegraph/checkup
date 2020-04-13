package fs

import (
	"fmt"

	"github.com/sourcegraph/checkup/types"
)

const IndexName = "index.json"

// FilenameFormatString is the format string used
// by GenerateFilename to create a filename.
const FilenameFormatString = "%d-check.json"

// GenerateFilename returns a filename that is ideal
// for storing the results file on a storage provider
// that relies on the filename for retrieval that is
// sorted by date/timeframe. It returns a string pointer
// to be used by the AWS SDK...
func GenerateFilename() *string {
	s := fmt.Sprintf(FilenameFormatString, types.Timestamp())
	return &s
}
