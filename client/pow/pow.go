package pow

import (
	"crypto/sha1"
	"encoding/base64"
	"math/rand"
)

func Generate(zeros uint8) string {
	n := 12
	id := make([]byte, n+4)
	for i := 0; i < n; i++ {
		id[i] = byte(rand.Intn(256))
	}

	var counter uint32 = 0
	hasher := sha1.New()
	for {
		for i := 0; i < 4; i++ {
			id[n+3-i] = byte((counter >> (8 * i)) % 256)
		}

		if check(zeros, hasher.Sum(id)) {
			return base64.StdEncoding.EncodeToString(id)
		}

		counter++
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
