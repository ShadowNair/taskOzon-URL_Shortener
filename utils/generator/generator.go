package generator

import (
	"crypto/rand"
	"fmt"
)

const Alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"

type RandomGenerator struct{}

func (r *RandomGenerator) Generate(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("length must be positive")
	}

	const alphaLen = byte(len(Alphabet))
	const maxrb = byte(255 - (256 % len(Alphabet)))

	out := make([]byte, n)
	buf := make([]byte, n*2)
	pos := 0

	for pos < n {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("read random bytes: %w", err)
		}

		for _, b := range buf {
			if b > maxrb {
				continue
			}
			out[pos] = Alphabet[b%alphaLen]
			pos++
			if pos == n {
				break
			}
		}
	}

	return string(out), nil
}