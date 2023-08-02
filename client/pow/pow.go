package pow

import (
	"crypto/sha1"
	"encoding/base64"
	"github.com/pkg/errors"
	"math/rand"
)

func Generate(zeros uint8) (string, error) {
	id := make([]byte, 16)
	hasher := sha1.New()
	n := 1 << 25
	for i := 0; i < n; i++ {
		if _, err := rand.Read(id); err != nil {
			return "", errors.Wrap(err, "failed to generate random ID")
		}

		if check(zeros, hasher.Sum(id)) {
			return base64.StdEncoding.EncodeToString(id), nil
		}
	}

	return "", errors.New("PoW ran out of time")
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
