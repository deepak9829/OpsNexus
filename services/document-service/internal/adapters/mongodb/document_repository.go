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

const (
	documentsCollection = "documents"
	versionsCollection  = "document_versions"
)

type documentRepository struct {
	db *mongo.Database
}

func NewDocumentRepository(db *mongo.Database) ports.DocumentRepository {
	return &documentRepository{db: db}
}

func (r *documentRepository) col() *mongo.Collection {
	return r.db.Collection(documentsCollection)
}

func (r *documentRepository) verCol() *mongo.Collection {
	return r.db.Collection(versionsCollection)
}

func (r *documentRepository) EnsureIndexes(ctx context.Context) error {
	docIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "tenant_id", Value: 1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "case_id", Value: 1}}},
	}
	if _, err := r.col().Indexes().CreateMany(ctx, docIndexes); err != nil {
		return err
	}

	verIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "document_id", Value: 1}}},
	}
	_, err := r.verCol().Indexes().CreateMany(ctx, verIndexes)
	return err
}

func (r *documentRepository) Create(ctx context.Context, doc *domain.Document) error {
	_, err := r.col().InsertOne(ctx, doc)
	return err
}

func (r *documentRepository) FindByID(ctx context.Context, id string) (*domain.Document, error) {
	var doc domain.Document
	err := r.col().FindOne(ctx, bson.M{"id": id}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrDocumentNotFound
	}
	return &doc, err
}

func (r *documentRepository) ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.Document, int64, error) {
	filter := bson.M{"tenant_id": tenantID}
	skip := int64((page - 1) * limit)

	total, err := r.col().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().SetSkip(skip).SetLimit(int64(limit)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.col().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var docs []*domain.Document
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, 0, err
	}
	return docs, total, nil
}

func (r *documentRepository) Delete(ctx context.Context, id string) error {
	result, err := r.col().DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return domain.ErrDocumentNotFound
	}
	return nil
}

func (r *documentRepository) AddVersion(ctx context.Context, version *domain.DocumentVersion) error {
	_, err := r.verCol().InsertOne(ctx, version)
	if err != nil {
		return err
	}
	_, err = r.col().UpdateOne(ctx,
		bson.M{"id": version.DocumentID},
		bson.M{
			"$set": bson.M{
				"current_version": version.Version,
				"updated_at":      time.Now().UTC(),
			},
			"$inc": bson.M{"version_count": 1},
		},
	)
	return err
}

func (r *documentRepository) ListVersions(ctx context.Context, documentID string) ([]*domain.DocumentVersion, error) {
	opts := options.Find().SetSort(bson.D{{Key: "version", Value: 1}})
	cursor, err := r.verCol().Find(ctx, bson.M{"document_id": documentID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var versions []*domain.DocumentVersion
	if err := cursor.All(ctx, &versions); err != nil {
		return nil, err
	}
	return versions, nil
}
