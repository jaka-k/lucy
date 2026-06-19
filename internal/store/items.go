package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
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
