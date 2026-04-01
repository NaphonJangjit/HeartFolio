package model

import "go.mongodb.org/mongo-driver/v2/bson"

type User struct {
	ID       bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Email    string        `bson:"email" json:"email"`
	Password string        `bson:"password" json:"-"`
	Role     string        `bson:"role" json:"role"`
	EXP      int64         `bson:"exp" json:"exp"`
	Level    int32         `bson:"level" json:"level"`
}
