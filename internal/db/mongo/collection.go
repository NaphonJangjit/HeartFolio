package mongo

import (
    "context"
    "errors"
    "reflect"

    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"
)

type Collection struct {
    Col *mongo.Collection
}

func (c *Collection) InsertOne(ctx context.Context, doc interface{}) (interface{}, error) {
    v := reflect.ValueOf(doc)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }
    if v.Kind() == reflect.Struct {
        idField := v.FieldByName("ID")
        if idField.IsValid() && idField.Type() == reflect.TypeOf(bson.ObjectID{}) {
            if idField.IsZero() {
                idField.Set(reflect.ValueOf(bson.NewObjectID()))
            }
        }
    }
    res, err := c.Col.InsertOne(ctx, doc)
    if err != nil {
        return nil, err
    }
    return res.InsertedID, nil
}

func (c *Collection) FindByID(ctx context.Context, id interface{}, result interface{}) error {
    oid, err := toObjectID(id)
    if err != nil {
        return err
    }
    return c.Col.FindOne(ctx, bson.M{"_id": oid}).Decode(result)
}

func (c *Collection) FindOne(ctx context.Context, filter interface{}, result interface{}) error {
    return c.Col.FindOne(ctx, filter).Decode(result)
}

func (c *Collection) Find(ctx context.Context, filter interface{}, results interface{}) error {
    cursor, err := c.Col.Find(ctx, filter)
    if err != nil {
        return err
    }
    defer cursor.Close(ctx)
    return cursor.All(ctx, results)
}

func (c *Collection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
    return c.Col.UpdateOne(ctx, filter, update)
}

func (c *Collection) UpdateByID(ctx context.Context, id interface{}, update interface{}) (*mongo.UpdateResult, error) {
    oid, err := toObjectID(id)
    if err != nil {
        return nil, err
    }
    return c.Col.UpdateByID(ctx, oid, update)
}

func (c *Collection) DeleteOne(ctx context.Context, filter interface{}) (*mongo.DeleteResult, error) {
    return c.Col.DeleteOne(ctx, filter)
}

func (c *Collection) DeleteByID(ctx context.Context, id interface{}) (*mongo.DeleteResult, error) {
    oid, err := toObjectID(id)
    if err != nil {
        return nil, err
    }
    return c.Col.DeleteOne(ctx, bson.M{"_id": oid})
}

func (c *Collection) Count(ctx context.Context, filter interface{}) (int64, error) {
    return c.Col.CountDocuments(ctx, filter)
}

func toObjectID(id interface{}) (bson.ObjectID, error) {
    switch v := id.(type) {
    case string:
        return bson.ObjectIDFromHex(v)
    case bson.ObjectID:
        return v, nil
    default:
        return bson.NilObjectID, errors.New("invalid id type: must be string or ObjectID")
    }
}