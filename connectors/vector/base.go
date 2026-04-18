package vector

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/cannonball10/foundation/connectors/embedding"
	"github.com/cannonball10/foundation/schemas/vector"
	"github.com/oklog/ulid/v2"
)

type VectorConnector interface {
	Query(ctx context.Context, collection *string, query string, limit *int, opts ...vector.QueryOption) ([]*vector.QueryResult, error)
	Upsert(ctx context.Context, collection *string, vector *vector.Vector) error
	Delete(ctx context.Context, collection *string, id string) error

	BatchUpsert(ctx context.Context, collection *string, vectors []*vector.Vector) error
	BatchDelete(ctx context.Context, collection *string, ids []string) error

	CreateCollection(ctx context.Context, collection *string) error
	DeleteCollection(ctx context.Context, collection *string) error
	ListCollections(ctx context.Context) ([]string, error)
}

func DefaultVectorConnector(ctx context.Context, embeddingConnector embedding.EmbeddingConnector) (VectorConnector, error) {
	return DefaultQdrantConnector(ctx, nil, embeddingConnector)
}

// ULIDToInt converts a ULID string into a 128-bit integer (big-endian).
// This is fully reversible with IntToULID.
func ULIDToInt(s string) (*big.Int, error) {
	id, err := ulid.Parse(s)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(id[:]), nil
}

// IntToULID converts a 128-bit integer (big-endian) back into the exact ULID string.
// The input must be in the range [0, 2^128-1].
func IntToULID(n *big.Int) (string, error) {
	if n == nil {
		return "", errors.New("int is nil")
	}
	if n.Sign() < 0 {
		return "", errors.New("int must be non-negative")
	}
	b := n.Bytes()
	if len(b) > 16 {
		return "", errors.New("int is too large to fit into 128 bits")
	}
	var id ulid.ULID
	copy(id[16-len(b):], b) // left-pad with zeros
	return id.String(), nil
}

// ulidStringToUUIDString encodes ULID bytes as a UUID-formatted hex string.
// This is reversible and can be used as a Qdrant "uuid" point id.
func ulidStringToUUIDString(s string) (string, error) {
	id, err := ulid.Parse(s)
	if err != nil {
		return "", err
	}
	b := id[:]
	// 8-4-4-4-12 (bytes: 4-2-2-2-6)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func uuidStringToULIDString(u string) (string, error) {
	hexStr := strings.ReplaceAll(u, "-", "")
	if len(hexStr) != 32 {
		return "", errors.New("invalid uuid format for ULID decoding")
	}
	raw, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}
	if len(raw) != 16 {
		return "", errors.New("invalid uuid length for ULID decoding")
	}
	var id ulid.ULID
	copy(id[:], raw)
	return id.String(), nil
}
