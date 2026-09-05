package connector

import (
	"context"
	"fmt"
	"subslice/pkg/model"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConnector struct {
	client   *mongo.Client
	database *mongo.Database
}

func NewMongoConnector(uri string) (*MongoConnector, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongo: %w", err)
	}

	// Extract database name from connection string or default
	dbName := "subslice_staging"
	db := client.Database(dbName)

	return &MongoConnector{
		client:   client,
		database: db,
	}, nil
}

func (m *MongoConnector) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}

func (m *MongoConnector) DiscoverGraph() (*model.StorageGraph, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collections, err := m.database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list mongo collections: %w", err)
	}

	entitiesMap := make(map[string]bool)
	var relationships []model.Relationship

	for _, col := range collections {
		entitiesMap[col] = true

		var sample bson.M
		err := m.database.Collection(col).FindOne(ctx, bson.M{}).Decode(&sample)
		if err == nil {
			for field := range sample {
				if len(field) > 3 && field[len(field)-3:] == "_id" {
					refTarget := field[:len(field)-3] + "s"
					relationships = append(relationships, model.Relationship{
						ConstraintName: fmt.Sprintf("mongo_ref_%s_%s", col, field),
						ChildEntity:    col,
						ChildField:     field,
						ParentEntity:   refTarget,
						ParentField:    "id",
					})
				}
			}
		}
	}

	return &model.StorageGraph{
		Entities:      entitiesMap,
		Relationships: relationships,
	}, nil
}

func (m *MongoConnector) FetchRecords(entity string, column string, values []interface{}, limit int) ([]model.Record, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := m.database.Collection(entity)
	filter := bson.M{}
	if len(values) > 0 {
		filter = bson.M{column: bson.M{"$in": values}}
	}

	findOpts := options.Find()
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
	}

	cursor, err := collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed querying collection %s: %w", entity, err)
	}
	defer cursor.Close(ctx)

	var records []model.Record
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed decoding document in %s: %w", entity, err)
		}

		// Normalize MongoDB _id to string if present
		if idVal, ok := doc["_id"]; ok {
			doc["id"] = fmt.Sprintf("%v", idVal)
			delete(doc, "_id")
		}

		records = append(records, model.Record{
			EntityName: entity,
			Data:       doc,
		})
	}

	return records, nil
}

func (m *MongoConnector) WriteStream(entity string, records []model.Record) error {
	if len(records) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := m.database.Collection(entity)
	var writes []mongo.WriteModel

	for _, record := range records {
		filter := bson.M{}
		if id, ok := record.Data["id"]; ok {
			filter["id"] = id
		} else if tenantId, ok := record.Data["tenant_id"]; ok {
			filter["tenant_id"] = tenantId
		} else {
			filter = record.Data
		}

		update := bson.M{"$set": record.Data}
		setModel := mongo.NewUpdateOneModel().SetFilter(filter).SetUpsert(true).SetUpdate(update)
		writes = append(writes, setModel)
	}

	_, err := collection.BulkWrite(ctx, writes)
	if err != nil {
		return fmt.Errorf("failed bulk write to collection %s: %w", entity, err)
	}

	return nil
}
