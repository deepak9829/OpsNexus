package dynamodb

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/opsnexus/notification-service/internal/domain"
	"github.com/opsnexus/notification-service/internal/ports"
)

type auditItem struct {
	TenantID   string `dynamodbav:"tenantId"`
	ID         string `dynamodbav:"eventId"`
	ActorID    string `dynamodbav:"actorId"`
	ActorEmail string `dynamodbav:"actorEmail"`
	Action     string `dynamodbav:"action"`
	Resource   string `dynamodbav:"resource"`
	ResourceID string `dynamodbav:"resourceId"`
	OldValue   string `dynamodbav:"oldValue,omitempty"` // JSON encoded
	NewValue   string `dynamodbav:"newValue,omitempty"` // JSON encoded
	IPAddress  string `dynamodbav:"ipAddress"`
	UserAgent  string `dynamodbav:"userAgent"`
	Timestamp  string `dynamodbav:"timestamp"`
}

func toAuditItem(e *domain.AuditEvent) (*auditItem, error) {
	item := &auditItem{
		TenantID:   e.TenantID,
		ID:         e.ID,
		ActorID:    e.ActorID,
		ActorEmail: e.ActorEmail,
		Action:     e.Action,
		Resource:   e.Resource,
		ResourceID: e.ResourceID,
		IPAddress:  e.IPAddress,
		UserAgent:  e.UserAgent,
		Timestamp:  e.Timestamp.Format(time.RFC3339),
	}
	if e.OldValue != nil {
		b, err := json.Marshal(e.OldValue)
		if err != nil {
			return nil, err
		}
		item.OldValue = string(b)
	}
	if e.NewValue != nil {
		b, err := json.Marshal(e.NewValue)
		if err != nil {
			return nil, err
		}
		item.NewValue = string(b)
	}
	return item, nil
}

func fromAuditItem(item *auditItem) *domain.AuditEvent {
	e := &domain.AuditEvent{
		ID:         item.ID,
		TenantID:   item.TenantID,
		ActorID:    item.ActorID,
		ActorEmail: item.ActorEmail,
		Action:     item.Action,
		Resource:   item.Resource,
		ResourceID: item.ResourceID,
		IPAddress:  item.IPAddress,
		UserAgent:  item.UserAgent,
	}
	if t, err := time.Parse(time.RFC3339, item.Timestamp); err == nil {
		e.Timestamp = t
	}
	if item.OldValue != "" {
		_ = json.Unmarshal([]byte(item.OldValue), &e.OldValue)
	}
	if item.NewValue != "" {
		_ = json.Unmarshal([]byte(item.NewValue), &e.NewValue)
	}
	return e
}

type auditRepository struct {
	client *dynamodb.Client
	table  string
}

func NewAuditRepository(client *dynamodb.Client, table string) ports.AuditRepository {
	return &auditRepository{client: client, table: table}
}

func (r *auditRepository) Record(ctx context.Context, event *domain.AuditEvent) error {
	item, err := toAuditItem(event)
	if err != nil {
		return err
	}
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

func (r *auditRepository) FindByID(ctx context.Context, tenantID, id string) (*domain.AuditEvent, error) {
	key, err := attributevalue.MarshalMap(map[string]string{
		"tenantId": tenantID,
		"eventId":  id,
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
		return nil, domain.ErrAuditEventNotFound
	}

	var item auditItem
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return nil, err
	}
	return fromAuditItem(&item), nil
}

func (r *auditRepository) Query(ctx context.Context, tenantID string, filter ports.AuditFilter, page, limit int) ([]*domain.AuditEvent, int64, error) {
	exprValues := map[string]types.AttributeValue{
		":tid": &types.AttributeValueMemberS{Value: tenantID},
	}
	exprNames := map[string]string{}

	filterParts := ""
	if filter.ActorID != nil {
		filterParts += " AND actorId = :actorId"
		exprValues[":actorId"] = &types.AttributeValueMemberS{Value: *filter.ActorID}
	}
	if filter.Action != nil {
		filterParts += " AND #action = :action"
		exprValues[":action"] = &types.AttributeValueMemberS{Value: *filter.Action}
		exprNames["#action"] = "action"
	}
	if filter.Resource != nil {
		filterParts += " AND #resource = :resource"
		exprValues[":resource"] = &types.AttributeValueMemberS{Value: *filter.Resource}
		exprNames["#resource"] = "resource"
	}
	if filter.ResourceID != nil {
		filterParts += " AND resourceId = :resourceId"
		exprValues[":resourceId"] = &types.AttributeValueMemberS{Value: *filter.ResourceID}
	}
	if filter.From != nil {
		filterParts += " AND #ts >= :from"
		exprValues[":from"] = &types.AttributeValueMemberS{Value: filter.From.Format(time.RFC3339)}
		exprNames["#ts"] = "timestamp"
	}
	if filter.To != nil {
		if _, ok := exprNames["#ts"]; !ok {
			exprNames["#ts"] = "timestamp"
		}
		filterParts += " AND #ts <= :to"
		exprValues[":to"] = &types.AttributeValueMemberS{Value: filter.To.Format(time.RFC3339)}
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(r.table),
		KeyConditionExpression:    aws.String("tenantId = :tid"),
		ExpressionAttributeValues: exprValues,
	}

	if filterParts != "" {
		input.FilterExpression = aws.String(filterParts[5:]) // strip leading " AND "
	}
	if len(exprNames) > 0 {
		input.ExpressionAttributeNames = exprNames
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, 0, err
	}

	var events []*domain.AuditEvent
	for _, av := range result.Items {
		var item auditItem
		if err := attributevalue.UnmarshalMap(av, &item); err != nil {
			continue
		}
		events = append(events, fromAuditItem(&item))
	}

	total := int64(len(events))

	// Manual pagination
	start := (page - 1) * limit
	end := start + limit
	if start >= len(events) {
		return []*domain.AuditEvent{}, total, nil
	}
	if end > len(events) {
		end = len(events)
	}
	return events[start:end], total, nil
}
