// Package store manages the MongoDB connection and registry operations.
package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Store holds the live MongoDB client and the target database name.
type Store struct {
	client *mongo.Client
	db     string
}

// Connect opens a MongoDB client, verifies reachability with a ping, and
// returns a Store. Call Disconnect when done.
func Connect(ctx context.Context, uri, dbName string) (*Store, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	return &Store{client: client, db: dbName}, nil
}

// Disconnect closes the underlying MongoDB connection.
func (s *Store) Disconnect(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

func (s *Store) database() *mongo.Database {
	return s.client.Database(s.db)
}

// DropDatabase drops the entire database. Intended for test cleanup only.
func (s *Store) DropDatabase(ctx context.Context) error {
	return s.database().Drop(ctx)
}
