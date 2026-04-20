package game

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"
)

// RNG abstracts randomness so tests can inject a deterministic source.
// All engine decisions that depend on chance (role dealing, deck shuffles,
// initial president selection) go through this interface.
type RNG interface {
	// IntN returns a uniformly random integer in [0, n). Must panic
	// if n <= 0, matching math/rand semantics.
	IntN(n int) int
	// Shuffle reorders a slice of length n in place using the Fisher-Yates
	// algorithm. swap is called with index pairs that should exchange.
	Shuffle(n int, swap func(i, j int))
	// Float64 returns a uniformly random float in [0, 1). Used by
	// probabilistic paths like the Cable Phase silence roll.
	Float64() float64
}

// CryptoRNG is a crypto/rand-backed RNG suitable for production. For
// Replicant randomness determines secret information (roles, policy
// order) so a CSPRNG is the right default.
type CryptoRNG struct{}

// NewCryptoRNG returns a ready-to-use CryptoRNG.
func NewCryptoRNG() CryptoRNG { return CryptoRNG{} }

// IntN implements RNG.
func (CryptoRNG) IntN(n int) int {
	if n <= 0 {
		panic("game: RNG.IntN requires n > 0")
	}
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		// crypto/rand.Reader is not expected to fail; fall back to an
		// approximation that is still unbiased on well-sized n.
		var buf [8]byte
		_, _ = rand.Read(buf[:])
		return int(binary.BigEndian.Uint64(buf[:]) % uint64(n))
	}
	return int(v.Int64())
}

// Shuffle implements RNG with a Fisher-Yates shuffle.
func (r CryptoRNG) Shuffle(n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		j := r.IntN(i + 1)
		swap(i, j)
	}
}

// Float64 implements RNG using 53 bits of CSPRNG output.
func (CryptoRNG) Float64() float64 {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	// Discard 11 bits to land in [0, 2^53) then divide to [0, 1).
	return float64(binary.BigEndian.Uint64(buf[:])>>11) / (1 << 53)
}

// SeededRNG is a deterministic RNG for tests. It uses a simple linear
// congruential generator; not suitable for production.
type SeededRNG struct {
	State uint64
}

// NewSeededRNG returns a deterministic RNG seeded with seed.
func NewSeededRNG(seed uint64) *SeededRNG { return &SeededRNG{State: seed | 1} }

// IntN implements RNG.
func (r *SeededRNG) IntN(n int) int {
	if n <= 0 {
		panic("game: SeededRNG.IntN requires n > 0")
	}
	r.State = r.State*6364136223846793005 + 1442695040888963407
	return int(r.State>>33) % n
}

// Shuffle implements RNG.
func (r *SeededRNG) Shuffle(n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		j := r.IntN(i + 1)
		swap(i, j)
	}
}

// Float64 implements RNG. Uses the same LCG state as IntN.
func (r *SeededRNG) Float64() float64 {
	r.State = r.State*6364136223846793005 + 1442695040888963407
	// Take upper 53 bits and normalise.
	return float64(r.State>>11) / (1 << 53)
}
