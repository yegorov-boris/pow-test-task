package pow

import (
	"crypto/sha1"
	"encoding/base64"
	"math/rand"
)

func Generate(zeros uint8) string {
	initialLength := 13
	id := make([]byte, initialLength)
	for i := 0; i < initialLength-1; i++ {
		id[i] = byte(rand.Intn(256))
	}

	id[initialLength] = 0
	hasher := sha1.New()
	for {
		if check(zeros, hasher.Sum(id)) {
			return base64.StdEncoding.EncodeToString(id)
		}
		if id[len(id)] == 255 {
			id = append(id, 0)
		} else {
			id[len(id)]++
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
