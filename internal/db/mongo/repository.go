package mongo

import (
    "context"
    "errors"

    "go.mongodb.org/mongo-driver/v2/mongo"
)

type Repository[T any] struct {
    coll *Collection
}

func NewRepository[T any](coll *Collection) *Repository[T] {
    return &Repository[T]{coll: coll}
}

func (r *Repository[T]) InsertOne(ctx context.Context, doc *T) (interface{}, error) {
    return r.coll.InsertOne(ctx, doc)
}

func (r *Repository[T]) FindByID(ctx context.Context, id interface{}) (*T, error) {
    var result T
    err := r.coll.FindByID(ctx, id, &result)
    if errors.Is(err, mongo.ErrNoDocuments) {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &result, nil
}

func (r *Repository[T]) FindOne(ctx context.Context, filter interface{}) (*T, error) {
    var result T
    err := r.coll.FindOne(ctx, filter, &result)
    if errors.Is(err, mongo.ErrNoDocuments) {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &result, nil
}

func (r *Repository[T]) Find(ctx context.Context, filter interface{}) ([]*T, error) {
    var results []*T
    err := r.coll.Find(ctx, filter, &results)
    return results, err
}

func (r *Repository[T]) UpdateByID(ctx context.Context, id interface{}, update interface{}) error {
    _, err := r.coll.UpdateByID(ctx, id, update)
    return err
}

func (r *Repository[T]) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
    return r.coll.Col.UpdateOne(ctx, filter, update)
}

func (r *Repository[T]) DeleteByID(ctx context.Context, id interface{}) error {
    _, err := r.coll.DeleteByID(ctx, id)
    return err
}

func (r *Repository[T]) DeleteOne(ctx context.Context, filter interface{}) (*mongo.DeleteResult, error) {
    return r.coll.Col.DeleteOne(ctx, filter)
}