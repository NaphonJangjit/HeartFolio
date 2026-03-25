package mongo

import "go.mongodb.org/mongo-driver/v2/mongo"

type Database struct {
    db *mongo.Database
}

func (db *Database) Collection(name string) *Collection {
    return &Collection{Col: db.db.Collection(name)}
}