package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/opsnexus/notification-service/internal/domain"
	"github.com/opsnexus/notification-service/internal/ports"
)

type notificationItem struct {
	TenantID  string            `dynamodbav:"tenantId"`
	ID        string            `dynamodbav:"notificationId"`
	UserID    string            `dynamodbav:"userId"`
	Type      string            `dynamodbav:"type"`
	Title     string            `dynamodbav:"title"`
	Body      string            `dynamodbav:"body"`
	Channel   string            `dynamodbav:"channel"`
	Read      bool              `dynamodbav:"read"`
	ReadAt    *string           `dynamodbav:"readAt,omitempty"`
	Metadata  map[string]string `dynamodbav:"metadata,omitempty"`
	CreatedAt string            `dynamodbav:"createdAt"`
}

func toNotificationItem(n *domain.Notification) *notificationItem {
	item := &notificationItem{
		TenantID:  n.TenantID,
		ID:        n.ID,
		UserID:    n.UserID,
		Type:      string(n.Type),
		Title:     n.Title,
		Body:      n.Body,
		Channel:   string(n.Channel),
		Read:      n.Read,
		Metadata:  n.Metadata,
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
	}
	if n.ReadAt != nil {
		s := n.ReadAt.Format(time.RFC3339)
		item.ReadAt = &s
	}
	return item
}

func fromNotificationItem(item *notificationItem) *domain.Notification {
	n := &domain.Notification{
		ID:       item.ID,
		TenantID: item.TenantID,
		UserID:   item.UserID,
		Type:     domain.NotificationType(item.Type),
		Title:    item.Title,
		Body:     item.Body,
		Channel:  domain.NotificationChannel(item.Channel),
		Read:     item.Read,
		Metadata: item.Metadata,
	}
	if t, err := time.Parse(time.RFC3339, item.CreatedAt); err == nil {
		n.CreatedAt = t
	}
	if item.ReadAt != nil {
		if t, err := time.Parse(time.RFC3339, *item.ReadAt); err == nil {
			n.ReadAt = &t
		}
	}
	return n
}

type notificationRepository struct {
	client *dynamodb.Client
	table  string
}

func NewNotificationRepository(client *dynamodb.Client, table string) ports.NotificationRepository {
	return &notificationRepository{client: client, table: table}
}

func (r *notificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	item := toNotificationItem(n)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.table),
		Item:      av,
	})
	return err
}

func (r *notificationRepository) FindByID(ctx context.Context, tenantID, id string) (*domain.Notification, error) {
	key, err := attributevalue.MarshalMap(map[string]string{
		"tenantId":       tenantID,
		"notificationId": id,
	})
	if err != nil {
		return nil, err
	}

	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.table),
		Key:       key,
	})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, domain.ErrNotificationNotFound
	}

	var item notificationItem
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return nil, err
	}
	return fromNotificationItem(&item), nil
}

func (r *notificationRepository) ListByUser(ctx context.Context, tenantID, userID string, page, limit int) ([]*domain.Notification, int64, error) {
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.table),
		KeyConditionExpression: aws.String("tenantId = :tid"),
		FilterExpression:       aws.String("userId = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":tid": &types.AttributeValueMemberS{Value: tenantID},
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	var notifications []*domain.Notification
	for _, av := range result.Items {
		var item notificationItem
		if err := attributevalue.UnmarshalMap(av, &item); err != nil {
			continue
		}
		notifications = append(notifications, fromNotificationItem(&item))
	}

	total := int64(len(notifications))

	// Manual pagination
	start := (page - 1) * limit
	end := start + limit
	if start >= len(notifications) {
		return []*domain.Notification{}, total, nil
	}
	if end > len(notifications) {
		end = len(notifications)
	}
	return notifications[start:end], total, nil
}

func (r *notificationRepository) MarkRead(ctx context.Context, tenantID, id string) error {
	key, err := attributevalue.MarshalMap(map[string]string{
		"tenantId":       tenantID,
		"notificationId": id,
	})
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        aws.String(r.table),
		Key:              key,
		UpdateExpression: aws.String("SET #r = :r, readAt = :ra"),
		ExpressionAttributeNames: map[string]string{
			"#r": "read",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":r":  &types.AttributeValueMemberBOOL{Value: true},
			":ra": &types.AttributeValueMemberS{Value: now},
		},
	})
	return err
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, tenantID, userID string) error {
	// Query all unread notifications for user
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.table),
		KeyConditionExpression: aws.String("tenantId = :tid"),
		FilterExpression:       aws.String("userId = :uid AND #r = :unread"),
		ExpressionAttributeNames: map[string]string{
			"#r": "read",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":tid":    &types.AttributeValueMemberS{Value: tenantID},
			":uid":    &types.AttributeValueMemberS{Value: userID},
			":unread": &types.AttributeValueMemberBOOL{Value: false},
		},
	})
	if err != nil {
		return err
	}

	for _, av := range result.Items {
		var item notificationItem
		if err := attributevalue.UnmarshalMap(av, &item); err != nil {
			continue
		}
		if err := r.MarkRead(ctx, tenantID, item.ID); err != nil {
			continue
		}
	}
	return nil
}

func (r *notificationRepository) Delete(ctx context.Context, tenantID, id string) error {
	key, err := attributevalue.MarshalMap(map[string]string{
		"tenantId":       tenantID,
		"notificationId": id,
	})
	if err != nil {
		return err
	}

	_, err = r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.table),
		Key:       key,
	})
	return err
}
