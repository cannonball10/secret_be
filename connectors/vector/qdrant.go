package vector

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"sync"

	"github.com/cannonball10/foundation/connectors/embedding"
	"github.com/cannonball10/foundation/schemas/vector"
	"github.com/oklog/ulid/v2"
	"github.com/qdrant/go-client/qdrant"
)

const (
	vectorSize   = 1536
	vectorType   = qdrant.Distance_Cosine
	defaultLimit = 10
)

type QdrantConnector struct {
	client        *qdrant.Client
	clientOnce    sync.Once
	clientFactory func() *qdrant.Client

	embeddingConnector embedding.EmbeddingConnector
}

func NewQdrantConnector(client *qdrant.Client, embeddingConnector embedding.EmbeddingConnector) *QdrantConnector {
	return &QdrantConnector{client: client, embeddingConnector: embeddingConnector}
}

func NewQdrantConnectorLazy(factory func() *qdrant.Client) *QdrantConnector {
	return newQdrantConnector(nil, factory, nil)
}

func newQdrantConnector(client *qdrant.Client, factory func() *qdrant.Client, embeddingConnector embedding.EmbeddingConnector) *QdrantConnector {
	return &QdrantConnector{client: client, clientFactory: factory, embeddingConnector: embeddingConnector}
}

func DefaultQdrantConnector(ctx context.Context, cfg *qdrant.Config, embeddingConnector embedding.EmbeddingConnector) (*QdrantConnector, error) {
	if cfg == nil {
		host := os.Getenv("QDRANT_HOST")
		if host == "" {
			return nil, errors.New("QDRANT_HOST is not set")
		}
		port, err := strconv.Atoi(os.Getenv("QDRANT_PORT"))
		if err != nil {
			return nil, errors.New("QDRANT_PORT is not set")
		}
		cfg = &qdrant.Config{
			Host: host,
			Port: port,
		}
	}
	client, err := qdrant.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return NewQdrantConnector(client, embeddingConnector), nil
}

func float64ToFloat32(vector []float64) []float32 {
	result := make([]float32, len(vector))
	for i, v := range vector {
		result[i] = float32(v)
	}
	return result
}

func PayloadToMap(payload map[string]*qdrant.Value) map[string]any {
	out := make(map[string]any, len(payload))
	for k, v := range payload {
		out[k] = valueToAny(v)
	}
	return out
}

func ulidStringToPointID(s string) (*qdrant.PointId, error) {
	u, err := ulidStringToUUIDString(s)
	if err != nil {
		return nil, err
	}
	return &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: u}}, nil
}

func pointIDToULIDString(id *qdrant.PointId) (string, error) {
	if id == nil {
		return "", errors.New("point id is nil")
	}
	switch id.PointIdOptions.(type) {
	case *qdrant.PointId_Uuid:
		return uuidStringToULIDString(id.GetUuid())
	case *qdrant.PointId_Num:
		// Legacy support: older points were stored as numeric time-only ids (lossy).
		// We cannot recover the original ULID; return a stable ULID with zero entropy.
		num := id.GetNum()
		var out ulid.ULID
		// ULID time is 48-bit big-endian milliseconds.
		out[0] = byte(num >> 40)
		out[1] = byte(num >> 32)
		out[2] = byte(num >> 24)
		out[3] = byte(num >> 16)
		out[4] = byte(num >> 8)
		out[5] = byte(num)
		// remaining 10 bytes already zero
		return out.String(), nil
	default:
		return "", errors.New("unknown point id type")
	}
}

func valueToAny(v *qdrant.Value) any {
	switch kind := v.Kind.(type) {
	case *qdrant.Value_NullValue:
		return nil
	case *qdrant.Value_BoolValue:
		return kind.BoolValue
	case *qdrant.Value_IntegerValue:
		return kind.IntegerValue
	case *qdrant.Value_DoubleValue:
		return kind.DoubleValue
	case *qdrant.Value_StringValue:
		return kind.StringValue
	case *qdrant.Value_ListValue:
		arr := make([]any, 0, len(kind.ListValue.Values))
		for _, lv := range kind.ListValue.Values {
			arr = append(arr, valueToAny(lv))
		}
		return arr
	case *qdrant.Value_StructValue:
		m := make(map[string]any, len(kind.StructValue.Fields))
		for k, sv := range kind.StructValue.Fields {
			m[k] = valueToAny(sv)
		}
		return m
	default:
		return nil
	}
}

func (c *QdrantConnector) Query(ctx context.Context, collection *string, query string, limit *int, opts ...vector.QueryOption) ([]*vector.QueryResult, error) {
	if collection == nil {
		return nil, errors.New("collection is required")
	}
	if *collection == "" {
		return nil, errors.New("collection name is required")
	}
	query_embedding, err := c.embeddingConnector.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	effectiveLimit := defaultLimit
	if limit != nil && *limit > 0 {
		effectiveLimit = *limit
	}
	query_vector := float64ToFloat32(query_embedding)
	req := &qdrant.SearchPoints{
		CollectionName: *collection,
		Vector:         query_vector,
		Limit:          uint64(effectiveLimit),
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	}
	for _, opt := range opts {
		switch opt := opt.(type) {
		case *vector.QueryScoreThresholdOption:
			scoreThreshold := float32(opt.ScoreThreshold)
			req.ScoreThreshold = &scoreThreshold
		case *vector.QueryFilterOption:
			if opt.Filter != nil {
				req.Filter = buildQdrantFilter(opt.Filter)
			}
		}
	}
	resp, err := c.client.GetPointsClient().Search(ctx, req)
	if err != nil {
		slog.ErrorContext(ctx, "qdrant Query failed", "collection", *collection, "limit", effectiveLimit, "error", err)
		return nil, err
	}
	results := make([]*vector.QueryResult, 0, len(resp.Result))
	for _, point := range resp.Result {
		id, err := pointIDToULIDString(point.Id)
		if err != nil {
			return nil, err
		}
		results = append(results, &vector.QueryResult{
			ID:      id,
			Score:   float64(point.Score),
			Payload: PayloadToMap(point.Payload),
		})
	}
	return results, nil
}

func (c *QdrantConnector) Upsert(ctx context.Context, collection *string, v *vector.Vector) error {
	return c.BatchUpsert(ctx, collection, []*vector.Vector{v})
}

func (c *QdrantConnector) BatchUpsert(ctx context.Context, collection *string, vectors []*vector.Vector) error {
	if collection == nil {
		return errors.New("collection is required")
	}
	if *collection == "" {
		return errors.New("collection name is required")
	}

	points := make([]*qdrant.PointStruct, 0, len(vectors))
	for _, vector := range vectors {
		pid, err := ulidStringToPointID(vector.ID)
		if err != nil {
			return err
		}
		points = append(points, &qdrant.PointStruct{
			Id:      pid,
			Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: qdrant.NewVector(float64ToFloat32(vector.Vector)...)}},
			Payload: qdrant.NewValueMap(vector.Payload),
		})
	}
	req := &qdrant.UpsertPoints{
		CollectionName: *collection,
		Points:         points,
	}

	_, err := c.client.Upsert(ctx, req)
	if err != nil {
		slog.ErrorContext(ctx, "qdrant BatchUpsert failed", "collection", *collection, "point_count", len(points), "error", err)
	}
	return err
}

func (c *QdrantConnector) Delete(ctx context.Context, collection *string, id string) error {
	return c.BatchDelete(ctx, collection, []string{id})
}

func (c *QdrantConnector) BatchDelete(ctx context.Context, collection *string, ids []string) error {
	if collection == nil {
		return errors.New("collection is required")
	}
	if *collection == "" {
		return errors.New("collection name is required")
	}
	point_ids := make([]*qdrant.PointId, 0, len(ids))
	for _, id := range ids {
		pid, err := ulidStringToPointID(id)
		if err != nil {
			return err
		}
		point_ids = append(point_ids, pid)
	}
	req := &qdrant.DeletePoints{
		CollectionName: *collection,
		Points:         &qdrant.PointsSelector{PointsSelectorOneOf: &qdrant.PointsSelector_Points{Points: &qdrant.PointsIdsList{Ids: point_ids}}},
	}
	_, err := c.client.Delete(ctx, req)
	if err != nil {
		slog.ErrorContext(ctx, "qdrant BatchDelete failed", "collection", *collection, "point_count", len(ids), "error", err)
	}
	return err
}

func (c *QdrantConnector) CreateCollection(ctx context.Context, collection *string) error {
	if collection == nil {
		return errors.New("collection is required")
	}
	if *collection == "" {
		return errors.New("collection name is required")
	}
	if err := c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: *collection,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: vectorType,
		}),
	}); err != nil {
		slog.ErrorContext(ctx, "qdrant CreateCollection failed", "collection", *collection, "error", err)
		return err
	}
	return nil
}

func (c *QdrantConnector) DeleteCollection(ctx context.Context, collection *string) error {
	if collection == nil {
		return errors.New("collection is required")
	}
	if *collection == "" {
		return errors.New("collection name is required")
	}
	if err := c.client.DeleteCollection(ctx, *collection); err != nil {
		slog.ErrorContext(ctx, "qdrant DeleteCollection failed", "collection", *collection, "error", err)
		return err
	}
	return nil
}

func (c *QdrantConnector) ListCollections(ctx context.Context) ([]string, error) {
	return c.client.ListCollections(ctx)
}

// Close closes the Qdrant client connection.
func (c *QdrantConnector) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// buildQdrantFilter converts a vector.Filter to a Qdrant filter.
func buildQdrantFilter(f *vector.Filter) *qdrant.Filter {
	if f == nil {
		return nil
	}

	filter := &qdrant.Filter{}

	if len(f.Must) > 0 {
		filter.Must = make([]*qdrant.Condition, 0, len(f.Must))
		for _, cond := range f.Must {
			if qCond := buildQdrantCondition(&cond); qCond != nil {
				filter.Must = append(filter.Must, qCond)
			}
		}
	}

	if len(f.Should) > 0 {
		filter.Should = make([]*qdrant.Condition, 0, len(f.Should))
		for _, cond := range f.Should {
			if qCond := buildQdrantCondition(&cond); qCond != nil {
				filter.Should = append(filter.Should, qCond)
			}
		}
	}

	if len(f.MustNot) > 0 {
		filter.MustNot = make([]*qdrant.Condition, 0, len(f.MustNot))
		for _, cond := range f.MustNot {
			if qCond := buildQdrantCondition(&cond); qCond != nil {
				filter.MustNot = append(filter.MustNot, qCond)
			}
		}
	}

	return filter
}

// buildQdrantCondition converts a vector.Condition to a Qdrant condition.
func buildQdrantCondition(cond *vector.Condition) *qdrant.Condition {
	if cond == nil || cond.Field == "" {
		return nil
	}

	// Handle Match conditions
	if cond.Match != nil {
		if cond.Match.Keyword != "" {
			return &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: cond.Field,
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: cond.Match.Keyword,
							},
						},
					},
				},
			}
		}
		if cond.Match.Value != nil {
			switch v := cond.Match.Value.(type) {
			case string:
				return &qdrant.Condition{
					ConditionOneOf: &qdrant.Condition_Field{
						Field: &qdrant.FieldCondition{
							Key: cond.Field,
							Match: &qdrant.Match{
								MatchValue: &qdrant.Match_Keyword{
									Keyword: v,
								},
							},
						},
					},
				}
			case int:
				return &qdrant.Condition{
					ConditionOneOf: &qdrant.Condition_Field{
						Field: &qdrant.FieldCondition{
							Key: cond.Field,
							Match: &qdrant.Match{
								MatchValue: &qdrant.Match_Integer{
									Integer: int64(v),
								},
							},
						},
					},
				}
			case int64:
				return &qdrant.Condition{
					ConditionOneOf: &qdrant.Condition_Field{
						Field: &qdrant.FieldCondition{
							Key: cond.Field,
							Match: &qdrant.Match{
								MatchValue: &qdrant.Match_Integer{
									Integer: v,
								},
							},
						},
					},
				}
			case bool:
				return &qdrant.Condition{
					ConditionOneOf: &qdrant.Condition_Field{
						Field: &qdrant.FieldCondition{
							Key: cond.Field,
							Match: &qdrant.Match{
								MatchValue: &qdrant.Match_Boolean{
									Boolean: v,
								},
							},
						},
					},
				}
			}
		}
	}

	// Handle Range conditions
	if cond.Range != nil {
		rangeFilter := &qdrant.Range{}
		if cond.Range.GT != nil {
			rangeFilter.Gt = cond.Range.GT
		}
		if cond.Range.GTE != nil {
			rangeFilter.Gte = cond.Range.GTE
		}
		if cond.Range.LT != nil {
			rangeFilter.Lt = cond.Range.LT
		}
		if cond.Range.LTE != nil {
			rangeFilter.Lte = cond.Range.LTE
		}
		return &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   cond.Field,
					Range: rangeFilter,
				},
			},
		}
	}

	return nil
}
