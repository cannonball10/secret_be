package vector

// QueryOption is a marker interface for query options.
type QueryOption any

// QueryScoreThresholdOption sets a minimum score threshold for results.
type QueryScoreThresholdOption struct {
	ScoreThreshold float64
}

// QueryFilterOption provides filtering support for vector queries.
type QueryFilterOption struct {
	Filter *Filter
}

// Filter represents a set of conditions for filtering vector search results.
type Filter struct {
	Must   []Condition `json:"must,omitempty"`
	Should []Condition `json:"should,omitempty"`
	MustNot []Condition `json:"must_not,omitempty"`
}

// Condition represents a single filter condition.
type Condition struct {
	Field string `json:"field"`
	Match *Match `json:"match,omitempty"`
	Range *Range `json:"range,omitempty"`
}

// Match represents an exact match condition.
type Match struct {
	Value   any    `json:"value,omitempty"`
	Keyword string `json:"keyword,omitempty"`
}

// Range represents a range condition for numeric fields.
type Range struct {
	GT  *float64 `json:"gt,omitempty"`
	GTE *float64 `json:"gte,omitempty"`
	LT  *float64 `json:"lt,omitempty"`
	LTE *float64 `json:"lte,omitempty"`
}
