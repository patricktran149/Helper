package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

func AggregateQuery(db *mongo.Database, table, query string, firstMatch bson.D, limit int) (bytes []byte, err error) {
	var (
		queryArrayM []bson.M
		pipeline    mongo.Pipeline
		result      []bson.M
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	if err = json.Unmarshal([]byte(query), &queryArrayM); err != nil {
		return bytes, errors.New("Unmarshal raw Query to Array Bson M ERROR - " + err.Error())
	}

	bsonDArray := ArrayBsonMToD(queryArrayM)

	bsonDArray = append([]bson.D{firstMatch}, bsonDArray...)

	if limit > 0 {
		bsonDArray = append(bsonDArray, bson.D{{"$limit", limit}})
	}

	pipeline = bsonDArray

	cursor, err := db.Collection(table).Aggregate(ctx, pipeline)
	if err != nil {
		return bytes, errors.New("Find ERROR - " + err.Error())
	}

	if err = cursor.All(ctx, &result); err != nil {
		return bytes, errors.New("Cursor All ERROR - " + err.Error())
	}

	bytes, err = json.Marshal(result)
	if err != nil {
		return bytes, errors.New("Marshal result ERROR - " + err.Error())
	}

	return bytes, nil
}

func ArrayBsonMToD(arrayM []bson.M) (arrayD []bson.D) {
	for _, doc := range arrayM {
		arrayD = append(arrayD, BsonMToD(doc))
	}

	return arrayD
}

func BsonMToD(m bson.M) (d bson.D) {
	for key, value := range m {
		d = append(d, bson.E{Key: key, Value: value})
	}
	return d
}
