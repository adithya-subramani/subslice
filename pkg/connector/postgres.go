package connector

import (
	"database/sql"
	"fmt"
	"strings"
	"subslice/pkg/model"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresConnector struct {
	db *sql.DB
}

func NewPostgresConnector(connStr string) (*PostgresConnector, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres host: %w", err)
	}

	return &PostgresConnector{db: db}, nil
}

func (p *PostgresConnector) Close() error {
	return p.db.Close()
}

func (p *PostgresConnector) DiscoverGraph() (*model.StorageGraph, error) {
	graph := &model.StorageGraph{
		Entities:      make(map[string]bool),
		Relationships: []model.Relationship{},
	}

	fkQuery := `
		SELECT
			tc.constraint_name,
			kcu.table_name AS child_table,
			kcu.column_name AS child_column,
			ccu.table_name AS parent_table,
			ccu.column_name AS parent_column
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
		  ON ccu.constraint_name = tc.constraint_name
		 AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = 'public';
	`

	rows, err := p.db.Query(fkQuery)
	if err != nil {
		return nil, fmt.Errorf("failed querying postgres foreign keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rel model.Relationship
		if err := rows.Scan(&rel.ConstraintName, &rel.ChildEntity, &rel.ChildField, &rel.ParentEntity, &rel.ParentField); err != nil {
			return nil, fmt.Errorf("failed scanning relationship row: %w", err)
		}

		graph.Entities[rel.ChildEntity] = true
		graph.Entities[rel.ParentEntity] = true
		graph.Relationships = append(graph.Relationships, rel)
	}

	return graph, nil
}

func (p *PostgresConnector) FetchRecords(entity string, field string, values []interface{}) ([]model.Record, error) {
	if len(values) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", entity, field, strings.Join(placeholders, ","))
	rows, err := p.db.Query(query, values...)
	if err != nil {
		return nil, fmt.Errorf("error querying entity %s: %w", entity, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var records []model.Record
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		recordData := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			recordData[colName] = *val
		}

		records = append(records, model.Record{
			EntityName: entity,
			Data:       recordData,
		})
	}

	return records, nil
}

func (p *PostgresConnector) WriteStream(entity string, records []model.Record) error {
	// Dummy streaming implementation for iteration 2
	return nil
}
