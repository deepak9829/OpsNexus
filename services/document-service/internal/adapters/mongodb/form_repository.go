package mongodb

import (
	"context"
	"time"

	"github.com/opsnexus/document-service/internal/domain"
	"github.com/opsnexus/document-service/internal/ports"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const formsCollection = "forms"

type formTemplateRepository struct {
	db *mongo.Database
}

func NewFormTemplateRepository(db *mongo.Database) ports.FormTemplateRepository {
	return &formTemplateRepository{db: db}
}

func (r *formTemplateRepository) collection() *mongo.Collection {
	return r.db.Collection(formsCollection)
}

func (r *formTemplateRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "tenant_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "status", Value: 1}},
		},
	}
	_, err := r.collection().Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *formTemplateRepository) Create(ctx context.Context, form *domain.FormTemplate) error {
	_, err := r.collection().InsertOne(ctx, form)
	return err
}

func (r *formTemplateRepository) FindByID(ctx context.Context, id string) (*domain.FormTemplate, error) {
	var form domain.FormTemplate
	err := r.collection().FindOne(ctx, bson.M{"id": id}).Decode(&form)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrFormNotFound
	}
	return &form, err
}

func (r *formTemplateRepository) ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.FormTemplate, int64, error) {
	filter := bson.M{"tenant_id": tenantID}
	skip := int64((page - 1) * limit)

	total, err := r.collection().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().SetSkip(skip).SetLimit(int64(limit)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var forms []*domain.FormTemplate
	if err := cursor.All(ctx, &forms); err != nil {
		return nil, 0, err
	}
	return forms, total, nil
}

func (r *formTemplateRepository) Update(ctx context.Context, form *domain.FormTemplate) error {
	form.UpdatedAt = time.Now().UTC()
	_, err := r.collection().ReplaceOne(ctx, bson.M{"id": form.ID}, form)
	return err
}

func (r *formTemplateRepository) UpdateStatus(ctx context.Context, id string, status domain.FormStatus) error {
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now().UTC(),
		},
	}
	result, err := r.collection().UpdateOne(ctx, bson.M{"id": id}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrFormNotFound
	}
	return nil
}
