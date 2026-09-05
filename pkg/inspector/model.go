package inspector

// ForeignKey represents a directed constraint from a child table to a parent table.
type ForeignKey struct {
	ConstraintName string
	ChildTable     string
	ChildColumn    string
	ParentTable    string
	ParentColumn   string
}

// TableSchema holds table metadata including primary keys and foreign keys.
type TableSchema struct {
	TableName   string
	PrimaryKey  string
	ForeignKeys []ForeignKey
}

// SchemaGraph contains all discovered table relationships across the database.
type SchemaGraph struct {
	Tables      map[string]*TableSchema
	ForeignKeys []ForeignKey
}
