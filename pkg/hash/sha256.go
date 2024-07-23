package hash

import "crypto/sha256"

func GetHashSHA256(key string, data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	h.Write([]byte(key))
	return h.Sum(nil)
}
