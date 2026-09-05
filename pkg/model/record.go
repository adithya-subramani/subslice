package model

// Record represents a single row, document, or key-value entry in a storage-agnostic format.
type Record struct {
	EntityName string                 // e.g., "users", "orders", or Mongo collection name
	PKValue    string                 // String representation of primary key / ID
	Data       map[string]interface{} // Field-value pairs
}

// Relationship represents a link between two entities (relational FK or logical NoSQL link)
type Relationship struct {
	ConstraintName string
	ChildEntity    string
	ChildField     string
	ParentEntity   string
	ParentField    string
}

// StorageGraph contains the full schema/relationship topology
type StorageGraph struct {
	Entities      map[string]bool
	Relationships []Relationship
}
