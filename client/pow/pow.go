package pow

import (
	"crypto/sha1"
	"encoding/base64"
	"math/rand"
)

func Generate(zeros uint8) string {
	n := 12
	id := make([]byte, n+1)
	for i := 0; i < n; i++ {
		id[i] = byte(rand.Intn(256))
	}

	id[n] = 0
	hasher := sha1.New()
	for {
		if check(zeros, hasher.Sum(id)) {
			return base64.StdEncoding.EncodeToString(id)
		}
		if id[n] == 255 {
			id = append(id, 0)
			n++
		} else {
			id[n]++
		}
	}
}

func check(zeros uint8, hashSum []byte) bool {
	fullyZeroBytes := int(zeros / 8)
	if len(hashSum) < fullyZeroBytes+1 {
		return false
	}

	for i := 0; i < fullyZeroBytes; i++ {
		if hashSum[i] != 0 {
			return false
		}
	}

	lastZeros := zeros % 8

	return lastZeros == 0 || (hashSum[fullyZeroBytes]>>(8-lastZeros)) == 0
}
