package logics

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"semo-server/configs"
	"semo-server/internal/repositories"
)

// Question represents the structure of a question document in MongoDB
type Question struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Question  string             `bson:"question"`
	Answer    string             `bson:"answer,omitempty"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

// MapQuestionService handles operations related to questions in MongoDB
type MapQuestionService struct {
	collection *mongo.Collection
}

// NewMapQuestionService creates a new instance of MapQuestionService
func NewMapQuestionService() *MapQuestionService {
	// Get the MongoDB client from repositories
	client := repositories.DBS.MongoDB

	// Get the database and collection
	db := client.Database("semo")
	collection := db.Collection("questions")

	return &MapQuestionService{
		collection: collection,
	}
}

// CreateQuestion creates a new question document in MongoDB and returns its ObjectID
func (s *MapQuestionService) CreateQuestion(question string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a new question document
	now := time.Now()
	newQuestion := Question{
		Question:  question,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Insert the document into MongoDB
	result, err := s.collection.InsertOne(ctx, newQuestion)
	if err != nil {
		configs.Logger.Error("Failed to insert question",
			zap.String("question", question),
			zap.Error(err))
		return "", err
	}

	// Extract the ID from the result
	id, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		configs.Logger.Error("Failed to convert InsertedID to ObjectID")
		return "", errors.New("failed to convert InsertedID to ObjectID")
	}

	configs.Logger.Info("Question created successfully",
		zap.String("id", id.Hex()),
		zap.String("question", question))

	return id.Hex(), nil
}

// GetQuestionByID retrieves a question document by its ObjectID
func (s *MapQuestionService) GetQuestionByID(id string) (*Question, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		configs.Logger.Error("Invalid ObjectID format",
			zap.String("id", id),
			zap.Error(err))
		return nil, err
	}

	// Create a filter to find the question by ID
	filter := bson.M{"_id": objectID}

	// Find the document
	var question Question
	err = s.collection.FindOne(ctx, filter).Decode(&question)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			configs.Logger.Warn("Question not found", zap.String("id", id))
			return nil, errors.New("question not found")
		}

		configs.Logger.Error("Failed to find question",
			zap.String("id", id),
			zap.Error(err))
		return nil, err
	}

	return &question, nil
}

// UpdateQuestionWithAnswer updates a question document with an answer
func (s *MapQuestionService) UpdateQuestionWithAnswer(id string, answer string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		configs.Logger.Error("Invalid ObjectID format",
			zap.String("id", id),
			zap.Error(err))
		return err
	}

	// Create a filter to find the question by ID
	filter := bson.M{"_id": objectID}

	// Create an update to set the answer and update timestamp
	update := bson.M{
		"$set": bson.M{
			"answer":     answer,
			"updated_at": time.Now(),
		},
	}

	// Update the document
	result, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		configs.Logger.Error("Failed to update question with answer",
			zap.String("id", id),
			zap.Error(err))
		return err
	}

	if result.MatchedCount == 0 {
		configs.Logger.Warn("Question not found for update", zap.String("id", id))
		return errors.New("question not found")
	}

	configs.Logger.Info("Question updated with answer successfully",
		zap.String("id", id),
		zap.Int64("modifiedCount", result.ModifiedCount))

	return nil
}
