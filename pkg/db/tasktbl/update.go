package tasktbl

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/kxplxn/goteam/pkg/db"
)

// Updater can be used to update a task in the task table.
type Updater struct{ ItemPutter db.DynamoItemPutter }

// NewUpdater creates and returns a new Updater.
func NewUpdater(ip db.DynamoItemPutter) Updater {
	return Updater{ItemPutter: ip}
}

// Update updates a task in the task table.
func (p Updater) Update(ctx context.Context, task Task) error {
	item, err := attributevalue.MarshalMap(task)
	if err != nil {
		return err
	}

	_, err = p.ItemPutter.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(os.Getenv(tableName)),
		Item:                item,
		ConditionExpression: aws.String("attribute_exists(ID)"),
	})

	var ex *types.ConditionalCheckFailedException
	if errors.As(err, &ex) {
		return db.ErrNoItem
	}

	return err
}
