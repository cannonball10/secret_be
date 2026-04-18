//go:build integration

package vector

import (
	"context"
	"hash/fnv"
	"math"
	"testing"

	"github.com/cannonball10/foundation/schemas/vector"
	"github.com/cannonball10/foundation/testutil"
	"github.com/cannonball10/foundation/utils"
	"github.com/qdrant/go-client/qdrant"
)

// MockEmbeddingConnector generates deterministic embeddings from text hashes.
// This allows testing vector operations without calling external APIs.
type MockEmbeddingConnector struct{}

func (m *MockEmbeddingConnector) Embed(ctx context.Context, payload any) ([]float64, error) {
	return generateDeterministicVector(payload), nil
}

func (m *MockEmbeddingConnector) EmbedBatch(ctx context.Context, payloads []any) ([][]float64, error) {
	results := make([][]float64, len(payloads))
	for i, payload := range payloads {
		results[i] = generateDeterministicVector(payload)
	}
	return results, nil
}

// generateDeterministicVector creates a normalized 1536-dimension vector from a hash.
func generateDeterministicVector(payload any) []float64 {
	// Convert payload to string for hashing
	var text string
	switch v := payload.(type) {
	case string:
		text = v
	default:
		text = "default"
	}

	h := fnv.New64a()
	h.Write([]byte(text))
	seed := h.Sum64()

	vec := make([]float64, 1536)
	var sumSquares float64

	// Generate pseudo-random values using LCG
	a := uint64(6364136223846793005)
	c := uint64(1442695040888963407)
	state := seed

	for i := 0; i < 1536; i++ {
		state = state*a + c
		// Normalize to [-1, 1]
		val := float64(int64(state)>>32) / float64(1<<31)
		vec[i] = val
		sumSquares += val * val
	}

	// Normalize the vector
	norm := math.Sqrt(sumSquares)
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}

	return vec
}

func setupQdrantTest(t *testing.T) (*QdrantConnector, string) {
	t.Helper()
	testutil.SkipIfServiceUnavailable(t, testutil.QdrantAddr, "Qdrant")

	cfg := &qdrant.Config{
		Host: "127.0.0.1",
		Port: 6334,
	}

	client, err := qdrant.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create Qdrant client: %v", err)
	}

	mockEmbedding := &MockEmbeddingConnector{}
	connector := NewQdrantConnector(client, mockEmbedding)
	prefix := testutil.TestPrefix(t)

	return connector, prefix
}

func createTestCollection(t *testing.T, connector *QdrantConnector, name string) {
	t.Helper()
	ctx := context.Background()

	err := connector.CreateCollection(ctx, &name)
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	t.Cleanup(func() {
		connector.DeleteCollection(ctx, &name)
	})
}

func TestQdrant_CreateDeleteCollection(t *testing.T) {
	connector, prefix := setupQdrantTest(t)
	ctx := context.Background()

	collectionName := prefix + "testcollection"

	// Create collection
	err := connector.CreateCollection(ctx, &collectionName)
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	// Verify collection exists
	collections, err := connector.ListCollections(ctx)
	if err != nil {
		t.Fatalf("ListCollections failed: %v", err)
	}

	found := false
	for _, c := range collections {
		if c == collectionName {
			found = true
			break
		}
	}
	if !found {
		t.Error("Created collection not found in list")
	}

	// Delete collection
	err = connector.DeleteCollection(ctx, &collectionName)
	if err != nil {
		t.Fatalf("DeleteCollection failed: %v", err)
	}

	// Verify collection is gone
	collections, err = connector.ListCollections(ctx)
	if err != nil {
		t.Fatalf("ListCollections after delete failed: %v", err)
	}

	for _, c := range collections {
		if c == collectionName {
			t.Error("Collection should not exist after delete")
		}
	}
}

func TestQdrant_ListCollections(t *testing.T) {
	connector, prefix := setupQdrantTest(t)
	ctx := context.Background()

	coll1 := prefix + "list1"
	coll2 := prefix + "list2"

	// Create two collections
	createTestCollection(t, connector, coll1)
	createTestCollection(t, connector, coll2)

	// List collections
	collections, err := connector.ListCollections(ctx)
	if err != nil {
		t.Fatalf("ListCollections failed: %v", err)
	}

	found1, found2 := false, false
	for _, c := range collections {
		if c == coll1 {
			found1 = true
		}
		if c == coll2 {
			found2 = true
		}
	}

	if !found1 {
		t.Errorf("Collection %s not found", coll1)
	}
	if !found2 {
		t.Errorf("Collection %s not found", coll2)
	}
}

func TestQdrant_UpsertQuery(t *testing.T) {
	connector, prefix := setupQdrantTest(t)
	ctx := context.Background()

	collectionName := prefix + "upsertquery"
	createTestCollection(t, connector, collectionName)

	// Create a vector
	id := utils.GenerateULID()
	text := "This is a test document about cats"
	embedding := generateDeterministicVector(text)

	vec := &vector.Vector{
		ID:      id,
		Vector:  embedding,
		Payload: map[string]any{"text": text, "category": "animals"},
	}

	// Upsert vector
	err := connector.Upsert(ctx, &collectionName, vec)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Query with same text (should find itself with high similarity)
	limit := 10
	results, err := connector.Query(ctx, &collectionName, text, &limit)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least 1 result")
	}

	if results[0].ID != id {
		t.Errorf("Expected ID %s, got %s", id, results[0].ID)
	}

	// Score should be very high for exact match
	if results[0].Score < 0.99 {
		t.Errorf("Expected high similarity score, got %f", results[0].Score)
	}
}

func TestQdrant_BatchUpsert(t *testing.T) {
	connector, prefix := setupQdrantTest(t)
	ctx := context.Background()

	collectionName := prefix + "batchupsert"
	createTestCollection(t, connector, collectionName)

	// Create multiple vectors
	vectors := make([]*vector.Vector, 3)
	texts := []string{"dogs are loyal", "cats are independent", "birds can fly"}

	for i, text := range texts {
		vectors[i] = &vector.Vector{
			ID:      utils.GenerateULID(),
			Vector:  generateDeterministicVector(text),
			Payload: map[string]any{"text": text, "index": i},
		}
	}

	// Batch upsert
	err := connector.BatchUpsert(ctx, &collectionName, vectors)
	if err != nil {
		t.Fatalf("BatchUpsert failed: %v", err)
	}

	// Query and verify all vectors are present
	limit := 10
	for i, text := range texts {
		results, err := connector.Query(ctx, &collectionName, text, &limit)
		if err != nil {
			t.Fatalf("Query for text %d failed: %v", i, err)
		}

		if len(results) == 0 {
			t.Errorf("No results for text %d", i)
			continue
		}

		// The exact text should return itself as the top result
		if results[0].ID != vectors[i].ID {
			t.Errorf("Text %d: expected ID %s as top result, got %s", i, vectors[i].ID, results[0].ID)
		}
	}
}

func TestQdrant_Delete(t *testing.T) {
	connector, prefix := setupQdrantTest(t)
	ctx := context.Background()

	collectionName := prefix + "delete"
	createTestCollection(t, connector, collectionName)

	// Create and insert a vector
	id := utils.GenerateULID()
	text := "delete me"
	vec := &vector.Vector{
		ID:      id,
		Vector:  generateDeterministicVector(text),
		Payload: map[string]any{"text": text},
	}

	err := connector.Upsert(ctx, &collectionName, vec)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Verify it exists
	limit := 10
	results, err := connector.Query(ctx, &collectionName, text, &limit)
	if err != nil {
		t.Fatalf("Query before delete failed: %v", err)
	}
	if len(results) == 0 || results[0].ID != id {
		t.Fatal("Vector should exist before delete")
	}

	// Delete vector
	err = connector.Delete(ctx, &collectionName, id)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	results, err = connector.Query(ctx, &collectionName, text, &limit)
	if err != nil {
		t.Fatalf("Query after delete failed: %v", err)
	}

	for _, r := range results {
		if r.ID == id {
			t.Error("Vector should not exist after delete")
		}
	}
}

func TestQdrant_BatchDelete(t *testing.T) {
	connector, prefix := setupQdrantTest(t)
	ctx := context.Background()

	collectionName := prefix + "batchdelete"
	createTestCollection(t, connector, collectionName)

	// Create and insert multiple vectors
	ids := make([]string, 3)
	vectors := make([]*vector.Vector, 3)
	texts := []string{"batch delete 1", "batch delete 2", "batch delete 3"}

	for i, text := range texts {
		ids[i] = utils.GenerateULID()
		vectors[i] = &vector.Vector{
			ID:      ids[i],
			Vector:  generateDeterministicVector(text),
			Payload: map[string]any{"text": text},
		}
	}

	err := connector.BatchUpsert(ctx, &collectionName, vectors)
	if err != nil {
		t.Fatalf("BatchUpsert failed: %v", err)
	}

	// Batch delete
	err = connector.BatchDelete(ctx, &collectionName, ids)
	if err != nil {
		t.Fatalf("BatchDelete failed: %v", err)
	}

	// Verify all are gone
	limit := 10
	for i, text := range texts {
		results, err := connector.Query(ctx, &collectionName, text, &limit)
		if err != nil {
			t.Fatalf("Query after delete failed: %v", err)
		}

		for _, r := range results {
			if r.ID == ids[i] {
				t.Errorf("Vector %d should not exist after batch delete", i)
			}
		}
	}
}

func TestQdrant_Query_WithLimit(t *testing.T) {
	connector, prefix := setupQdrantTest(t)
	ctx := context.Background()

	collectionName := prefix + "querylimit"
	createTestCollection(t, connector, collectionName)

	// Insert 5 vectors
	vectors := make([]*vector.Vector, 5)
	for i := 0; i < 5; i++ {
		text := "similar document about testing " + string(rune('A'+i))
		vectors[i] = &vector.Vector{
			ID:      utils.GenerateULID(),
			Vector:  generateDeterministicVector(text),
			Payload: map[string]any{"text": text, "index": i},
		}
	}

	err := connector.BatchUpsert(ctx, &collectionName, vectors)
	if err != nil {
		t.Fatalf("BatchUpsert failed: %v", err)
	}

	// Query with limit of 2
	limit := 2
	results, err := connector.Query(ctx, &collectionName, "testing documents", &limit)
	if err != nil {
		t.Fatalf("Query with limit failed: %v", err)
	}

	if len(results) > limit {
		t.Errorf("Expected at most %d results, got %d", limit, len(results))
	}
}

func TestQdrant_Query_WithScoreThreshold(t *testing.T) {
	connector, prefix := setupQdrantTest(t)
	ctx := context.Background()

	collectionName := prefix + "scorethreshold"
	createTestCollection(t, connector, collectionName)

	// Insert vectors with different content
	highSimilarity := "cats are wonderful pets"
	lowSimilarity := "quantum physics equations"

	vectors := []*vector.Vector{
		{
			ID:      utils.GenerateULID(),
			Vector:  generateDeterministicVector(highSimilarity),
			Payload: map[string]any{"text": highSimilarity},
		},
		{
			ID:      utils.GenerateULID(),
			Vector:  generateDeterministicVector(lowSimilarity),
			Payload: map[string]any{"text": lowSimilarity},
		},
	}

	err := connector.BatchUpsert(ctx, &collectionName, vectors)
	if err != nil {
		t.Fatalf("BatchUpsert failed: %v", err)
	}

	// Query with high score threshold
	limit := 10
	threshold := &vector.QueryScoreThresholdOption{ScoreThreshold: 0.9}
	results, err := connector.Query(ctx, &collectionName, highSimilarity, &limit, threshold)
	if err != nil {
		t.Fatalf("Query with score threshold failed: %v", err)
	}

	// With high threshold, only the exact match should be returned
	for _, r := range results {
		if r.Score < 0.9 {
			t.Errorf("Result with score %f should not pass threshold 0.9", r.Score)
		}
	}
}
