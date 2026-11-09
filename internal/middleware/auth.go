// Package middleware holds request-context helpers. The standalone app has a
// single local user and no authentication: every request acts as LocalUserID.
package middleware

import "context"

// LocalUserID is the only user of the standalone app.
const LocalUserID = "local"

// GetUserID returns the local user id (there is no login in the standalone app).
func GetUserID(context.Context) (string, error) {
	return LocalUserID, nil
}
