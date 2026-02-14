package db

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Database wraps a MongoDB client and provides access to a named database.
type Database struct {
	client   *mongo.Client
	database *mongo.Database
}

// Connect establishes a connection to MongoDB and pings to verify.
func Connect(ctx context.Context, uri, dbName string) (*Database, error) {
	if uri == "" {
		return nil, fmt.Errorf("MONGODB_URI is not set")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &Database{
		client:   client,
		database: client.Database(dbName),
	}, nil
}

// Collection returns a handle to the named collection.
func (d *Database) Collection(name string) *mongo.Collection {
	return d.database.Collection(name)
}

// Close disconnects the MongoDB client.
func (d *Database) Close(ctx context.Context) error {
	return d.client.Disconnect(ctx)
}
