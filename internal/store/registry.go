package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Collection is a row from the collections registry.
type Collection struct {
	ID        bson.ObjectID `bson:"_id"`
	Name      string        `bson:"name"`
	CreatedAt time.Time     `bson:"created_at"`
}

// Schema is a row from the schemas registry.
type Schema struct {
	ID           bson.ObjectID   `bson:"_id"`
	CollectionID bson.ObjectID   `bson:"collection_id"`
	Schema       json.RawMessage `bson:"schema"`
	UpdatedAt    time.Time       `bson:"updated_at"`
}

// EnsureIndexes creates the unique indexes on collections.name and
// schemas.collection_id. Safe to call on every startup.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	colls := s.database().Collection("collections")
	_, err := colls.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("collections index: %w", err)
	}

	schemas := s.database().Collection("schemas")
	_, err = schemas.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "collection_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("schemas index: %w", err)
	}

	return nil
}

// ListCollections returns all rows from the collections registry, sorted by name.
func (s *Store) ListCollections(ctx context.Context) ([]Collection, error) {
	cur, err := s.database().Collection("collections").Find(ctx, bson.D{},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []Collection
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCollection looks up a collection registry row by ObjectID.
// Returns nil, nil when no document matches.
func (s *Store) GetCollection(ctx context.Context, id bson.ObjectID) (*Collection, error) {
	var c Collection
	err := s.database().Collection("collections").
		FindOne(ctx, bson.D{{Key: "_id", Value: id}}).
		Decode(&c)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// EnsureCollection returns the ObjectID for the named collection, creating it
// if it doesn't exist yet.
func (s *Store) EnsureCollection(ctx context.Context, name string) (bson.ObjectID, error) {
	colls := s.database().Collection("collections")

	var existing Collection
	err := colls.FindOne(ctx, bson.D{{Key: "name", Value: name}}).Decode(&existing)
	if err == nil {
		return existing.ID, nil
	}
	if err != mongo.ErrNoDocuments {
		return bson.ObjectID{}, err
	}

	id := bson.NewObjectID()
	_, err = colls.InsertOne(ctx, bson.D{
		{Key: "_id", Value: id},
		{Key: "name", Value: name},
		{Key: "created_at", Value: time.Now().UTC()},
	})
	if err != nil {
		return bson.ObjectID{}, err
	}
	return id, nil
}

// GetSchema returns the stored item schema for the given collection, or nil if
// no schema has been saved yet.
func (s *Store) GetSchema(ctx context.Context, collectionID bson.ObjectID) (json.RawMessage, error) {
	var row Schema
	err := s.database().Collection("schemas").
		FindOne(ctx, bson.D{{Key: "collection_id", Value: collectionID}}).
		Decode(&row)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.Schema, nil
}

// UpsertSchema stores (or replaces) the item schema for the given collection.
func (s *Store) UpsertSchema(ctx context.Context, collectionID bson.ObjectID, schema json.RawMessage) error {
	_, err := s.database().Collection("schemas").UpdateOne(
		ctx,
		bson.D{{Key: "collection_id", Value: collectionID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "schema", Value: schema},
			{Key: "updated_at", Value: time.Now().UTC()},
		}}, {Key: "$setOnInsert", Value: bson.D{
			{Key: "_id", Value: bson.NewObjectID()},
			{Key: "collection_id", Value: collectionID},
		}}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}
