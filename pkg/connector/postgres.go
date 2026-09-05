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

func (p *PostgresConnector) FetchRecords(entity string, column string, values []interface{}, limit int) ([]model.Record, error) {
	if len(values) == 0 {
		query := fmt.Sprintf("SELECT * FROM %s", entity)
		if limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", limit)
		}
		rows, err := p.db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return p.scanRows(rows, entity)
	}

	placeholders := make([]string, len(values))
	args := make([]interface{}, len(values))
	for i, val := range values {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = val
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", entity, column, strings.Join(placeholders, ","))
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed fetching records for %s: %w", entity, err)
	}
	defer rows.Close()

	return p.scanRows(rows, entity)
}

func (p *PostgresConnector) WriteStream(entity string, records []model.Record) error {
	if len(records) == 0 {
		return nil
	}

	// 1. Collect column names from the first record
	var cols []string
	for col := range records[0].Data {
		cols = append(cols, col)
	}

	// 2. Build parameterized INSERT statement
	placeholders := make([]string, len(records))
	var args []interface{}
	argCount := 1

	for i, record := range records {
		var rowPlaceholders []string
		for _, col := range cols {
			rowPlaceholders = append(rowPlaceholders, fmt.Sprintf("$%d", argCount))
			args = append(args, record.Data[col])
			argCount++
		}
		placeholders[i] = fmt.Sprintf("(%s)", strings.Join(rowPlaceholders, ","))
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s ON CONFLICT DO NOTHING",
		entity,
		strings.Join(cols, ","),
		strings.Join(placeholders, ","),
	)

	_, err := p.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed inserting records into target entity %s: %w", entity, err)
	}

	return nil
}

// scanRows converts sql.Rows into a slice of generic model.Records
func (p *PostgresConnector) scanRows(rows *sql.Rows, entityName string) ([]model.Record, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var records []model.Record

	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		data := make(map[string]interface{})
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				data[col] = string(b)
			} else {
				data[col] = val
			}
		}

		records = append(records, model.Record{
			EntityName: entityName,
			Data:       data,
		})
	}

	return records, nil
}
