package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	UserId primitive.ObjectID `json:"user_id" bson:"user_id"`
	Name   string             `json:"name" bson:"name"`
	Email  string             `json:"email" bson:"email"`
	Avatar string             `json:"avatar" bson:"avatar"`
}
