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

const submissionsCollection = "form_submissions"

type formSubmissionRepository struct {
	db *mongo.Database
}

func NewFormSubmissionRepository(db *mongo.Database) ports.FormSubmissionRepository {
	return &formSubmissionRepository{db: db}
}

func (r *formSubmissionRepository) collection() *mongo.Collection {
	return r.db.Collection(submissionsCollection)
}

func (r *formSubmissionRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "tenant_id", Value: 1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "form_id", Value: 1}}},
	}
	_, err := r.collection().Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *formSubmissionRepository) Create(ctx context.Context, sub *domain.FormSubmission) error {
	_, err := r.collection().InsertOne(ctx, sub)
	return err
}

func (r *formSubmissionRepository) FindByID(ctx context.Context, id string) (*domain.FormSubmission, error) {
	var sub domain.FormSubmission
	err := r.collection().FindOne(ctx, bson.M{"id": id}).Decode(&sub)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrSubmissionNotFound
	}
	return &sub, err
}

func (r *formSubmissionRepository) ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.FormSubmission, int64, error) {
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

	var subs []*domain.FormSubmission
	if err := cursor.All(ctx, &subs); err != nil {
		return nil, 0, err
	}
	return subs, total, nil
}

func (r *formSubmissionRepository) ListByForm(ctx context.Context, formID string, page, limit int) ([]*domain.FormSubmission, int64, error) {
	filter := bson.M{"form_id": formID}
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

	var subs []*domain.FormSubmission
	if err := cursor.All(ctx, &subs); err != nil {
		return nil, 0, err
	}
	return subs, total, nil
}

func (r *formSubmissionRepository) UpdateStatus(ctx context.Context, id string, status domain.SubmissionStatus) error {
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
		return domain.ErrSubmissionNotFound
	}
	return nil
}
