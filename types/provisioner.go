package types

import (
	"fmt"
)

// ProvisionInfo contains the results of provisioning a new
// storage facility for check files. Its values should be
// used by the status page in order to obtain read-only
// access to the check files.
type ProvisionInfo struct {
	// The ID of a user that was created for accessing checks.
	UserID string `json:"user_id"`

	// The username of a user that was created for accessing checks.
	Username string `json:"username"`

	// The ID or name of the ID/key used to access checks. Expect
	// this value to be made public. (It should have read-only
	// access to the checks.)
	PublicAccessKeyID string `json:"public_access_key_id"`

	// The "secret" associated with the PublicAccessKeyID, but
	// expect this value to be made public. (It should provide
	// read-only access to the checks.)
	PublicAccessKey string `json:"public_access_key"`
}

// String returns the information in i in a human-readable format
// along with an important notice.
func (i ProvisionInfo) String() string {
	s := "Provision successful\n\n"
	s += fmt.Sprintf("             User ID: %s\n", i.UserID)
	s += fmt.Sprintf("            Username: %s\n", i.Username)
	s += fmt.Sprintf("Public Access Key ID: %s\n", i.PublicAccessKeyID)
	s += fmt.Sprintf("   Public Access Key: %s\n\n", i.PublicAccessKey)
	s += `IMPORTANT: Copy the Public Access Key ID and Public Access
Key into the config.js file for your status page. You will
not be shown these credentials again.`
	return s
}
