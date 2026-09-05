package connector

import (
	"fmt"
	"strings"
	"subslice/pkg/model"
)

// Connector abstracts schema discovery and record transport across SQL and NoSQL databases.
type Connector interface {
	// DiscoverGraph retrieves entities and explicit relationships from storage metadata
	DiscoverGraph() (*model.StorageGraph, error)

	// FetchRecords fetches matching records for an entity given field key/value pairs
	FetchRecords(entity string, field string, values []interface{}, limit int) ([]model.Record, error)

	// WriteStream streams batch records into the target destination
	WriteStream(entity string, records []model.Record) error

	// Close terminates open connections
	Close() error
}

// NewConnector acts as a factory constructing the appropriate Connector based on driver type
func NewConnector(driverType string, connStr string) (Connector, error) {
	switch strings.ToLower(driverType) {
	case "postgres", "postgresql", "cockroachdb", "cockroach":
		return NewPostgresConnector(connStr)
	case "mongo", "mongodb":
		return NewMongoConnector(connStr)
	default:
		return nil, fmt.Errorf("unsupported storage driver: %s", driverType)
	}
}
