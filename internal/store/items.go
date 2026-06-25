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

// InsertItems writes each item from the JSON array into the named data
// collection, stamping tag and created_at on every document.
// Returns the number of documents inserted.
func (s *Store) InsertItems(ctx context.Context, collectionName string, itemsJSON []byte, tag string) (int, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(itemsJSON, &raw); err != nil {
		return 0, fmt.Errorf("unmarshal items: %w", err)
	}
	if len(raw) == 0 {
		return 0, nil
	}

	now := time.Now().UTC()
	docs := make([]any, 0, len(raw))
	for _, item := range raw {
		var fields map[string]any
		if err := json.Unmarshal(item, &fields); err != nil {
			return 0, fmt.Errorf("unmarshal item: %w", err)
		}
		doc := bson.D{{Key: "_id", Value: bson.NewObjectID()}}
		for k, v := range fields {
			doc = append(doc, bson.E{Key: k, Value: v})
		}
		if tag != "" {
			doc = append(doc, bson.E{Key: "tag", Value: tag})
		}
		doc = append(doc, bson.E{Key: "created_at", Value: now})
		docs = append(docs, doc)
	}

	res, err := s.database().Collection(collectionName).InsertMany(ctx, docs)
	if err != nil {
		return 0, err
	}
	return len(res.InsertedIDs), nil
}

// ListItems returns the most recent documents from a data collection,
// newest first. When limit <= 0 it defaults to 100.
func (s *Store) ListItems(ctx context.Context, collectionName string, limit int64) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 100
	}
	cur, err := s.database().Collection(collectionName).Find(
		ctx, bson.D{},
		options.Find().
			SetSort(bson.D{{Key: "created_at", Value: -1}}).
			SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	out := make([]map[string]any, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteItem removes one document from the named collection.
// Returns mongo.ErrNoDocuments if nothing was deleted.
func (s *Store) DeleteItem(ctx context.Context, collectionName string, id bson.ObjectID) error {
	res, err := s.database().Collection(collectionName).
		DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// UpdateItem replaces the user-supplied fields of a document while preserving
// _id and created_at. The fields map may not include _id or created_at; both
// are stripped before applying the update.
func (s *Store) UpdateItem(ctx context.Context, collectionName string, id bson.ObjectID, fields map[string]any) error {
	delete(fields, "_id")
	delete(fields, "created_at")

	coll := s.database().Collection(collectionName)

	var existing bson.M
	if err := coll.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&existing); err != nil {
		return err
	}

	newDoc := bson.D{{Key: "_id", Value: id}}
	for k, v := range fields {
		newDoc = append(newDoc, bson.E{Key: k, Value: v})
	}
	if createdAt, ok := existing["created_at"]; ok {
		newDoc = append(newDoc, bson.E{Key: "created_at", Value: createdAt})
	}

	_, err := coll.ReplaceOne(ctx, bson.D{{Key: "_id", Value: id}}, newDoc)
	return err
}
