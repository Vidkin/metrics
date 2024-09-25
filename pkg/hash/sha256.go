// Package hash provides utilities for generating cryptographic hashes.
//
// This package includes functions to compute SHA-256 hashes, which can be used for
// data integrity verification, digital signatures, and other security-related tasks.
package hash

import "crypto/sha256"

// GetHashSHA256 computes the SHA-256 hash of the provided data combined with the specified key.
//
// Parameters:
//   - key: A string that acts as a key in the hashing process. It is appended to the
//     data before hashing, adding an additional layer of security and uniqueness to
//     the resulting hash output.
//   - data: A byte slice containing the data to be hashed. This is the main input
//     that will be combined with the key to generate the hash.
//
// Returns:
//   - A byte slice containing the SHA-256 hash of the concatenated data and key.
//     The length of the returned slice is always 32 bytes, as SHA-256 produces a
//     fixed-size output.
func GetHashSHA256(key string, data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	h.Write([]byte(key))
	return h.Sum(nil)
}
