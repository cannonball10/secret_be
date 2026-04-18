package graphdb

const (
	// DefaultUserLabel is the label used for user nodes in the graph database.
	DefaultUserLabel = "User"
	// DefaultUserIDProperty is the node property used as the unique user identifier.
	DefaultUserIDProperty = "userId"

	// DefaultRelationshipType is the (single) relationship type used to connect users.
	// Stored as a directed relationship, but matched as undirected for traversal.
	DefaultRelationshipType = "FRIEND"

	// DefaultBlockRelationshipType is a directed relationship type representing "A blocks B".
	DefaultBlockRelationshipType = "BLOCKS"
)
