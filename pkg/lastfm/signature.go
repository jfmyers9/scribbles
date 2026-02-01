package lastfm

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
)

// calculateSignature generates an MD5 signature for Last.fm API requests.
//
// The signature is calculated by:
// 1. Sorting parameter keys alphabetically
// 2. Concatenating key+value pairs (e.g., "keyAvalueAkeyBvalueB")
// 3. Appending the API secret
// 4. Taking the MD5 hash of the result
//
// This signature is required for all authenticated API requests.
func calculateSignature(params map[string]string, secret string) string {
	// Sort keys alphabetically
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build signature string: key1value1key2value2...secret
	var sigPlain string
	for _, k := range keys {
		sigPlain += k + params[k]
	}
	sigPlain += secret

	// Calculate MD5 hash
	hasher := md5.New()
	hasher.Write([]byte(sigPlain))
	return hex.EncodeToString(hasher.Sum(nil))
}
