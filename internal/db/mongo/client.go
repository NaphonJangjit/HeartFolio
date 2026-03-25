package mongo

import (
    "context"
    "log"
    "time"

    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Client struct {
    client *mongo.Client
}

func Connect(uri string) (*Client, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(options.Client().ApplyURI(uri))
    if err != nil {
        return nil, err
    }
    if err := client.Ping(ctx, nil); err != nil {
        return nil, err
    }
    log.Println("Connected to MongoDB")
    return &Client{client: client}, nil
}

func (c *Client) Close() {
    if c.client != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        if err := c.client.Disconnect(ctx); err != nil {
            log.Println("Error disconnecting MongoDB:", err)
        } else {
            log.Println("Disconnected from MongoDB")
        }
    }
}

func (c *Client) Database(name string) *Database {
    return &Database{db: c.client.Database(name)}
}