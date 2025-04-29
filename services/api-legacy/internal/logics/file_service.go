package logics

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"semo-server/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FileService provides functionality to upload files to S3 and generate download links.
type FileService struct {
	s3Client      *s3.Client
	bucketName    string
	presignClient *s3.PresignClient
	db            *gorm.DB
}

// NewFileService creates a new instance of FileService.
func NewFileService(s3Client *s3.Client, bucketName string, db *gorm.DB) *FileService {
	presignClient := s3.NewPresignClient(s3Client)
	return &FileService{
		s3Client:      s3Client,
		bucketName:    bucketName,
		presignClient: presignClient,
		db:            db,
	}
}

// UploadFile uploads the provided file (multipart.File and header) to S3 under the given itemID,
// and creates a new File record in the database.
func (fs *FileService) UploadFile(ctx context.Context, itemID string, file multipart.File, header *multipart.FileHeader) (*models.File, error) {
	// Ensure the file is closed when done.
	defer file.Close()

	// Generate a new UUID for the file.
	fileID := uuid.New()

	// Determine file extension from the original filename.
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != "" {
		ext = ext[1:] // remove the dot
	}

	// Define the S3 object key (예: "files/{fileID}")
	s3Key := fmt.Sprintf("files/%s/%s", itemID, fileID.String())

	// Prepare the PutObjectInput.
	putInput := &s3.PutObjectInput{
		Bucket:      aws.String(fs.bucketName),
		Key:         aws.String(s3Key),
		Body:        file, // file implements io.Reader
		ContentType: aws.String(header.Header.Get("Content-Type")),
		ACL:         s3types.ObjectCannedACLPrivate,
	}

	// Upload the file to S3.
	_, err := fs.s3Client.PutObject(ctx, putInput)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file to s3: %w", err)
	}

	// Create a new File record.
	newFile := models.File{
		ID:            fileID,
		FileName:      header.Filename,
		FileExtension: ext,
		ItemID:        itemID,
		ContentType:   header.Header.Get("Content-Type"),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := fs.db.Create(&newFile).Error; err != nil {
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	return &newFile, nil
}

// GetDownloadLink generates a presigned URL for downloading the file with the given fileID.
func (fs *FileService) GetDownloadLink(ctx context.Context, fileID uuid.UUID, itemID string) (string, error) {
	// Retrieve the file record from the database.
	var fileRecord models.File
	if err := fs.db.First(&fileRecord, "id = ?", fileID).Error; err != nil {
		return "", fmt.Errorf("failed to get file record: %w", err)
	}

	// Build the S3 object key (동일하게 "files/{fileID}"로 저장됨).
	s3Key := fmt.Sprintf("files/%s/%s", itemID, fileID.String())

	// Prepare the GetObjectInput.
	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(fs.bucketName),
		Key:    aws.String(s3Key),
	}

	// Generate a presigned URL with an expiration (예: 15분).
	presignResult, err := fs.presignClient.PresignGetObject(ctx, getObjectInput, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignResult.URL, nil
}

// ListFilesByItem retrieves all File records associated with the given itemID.
func (fs *FileService) ListFilesByItem(ctx context.Context, itemID string) ([]models.File, error) {
	var files []models.File
	if err := fs.db.Where("item_id = ?", itemID).Find(&files).Error; err != nil {
		return nil, fmt.Errorf("failed to list files for item %s: %w", itemID, err)
	}
	return files, nil
}

// DeleteFile deletes the file with the given fileID from S3 and removes its record from the database.
func (fs *FileService) DeleteFile(ctx context.Context, fileID uuid.UUID, itemID string) error {
	// 파일 레코드를 DB에서 조회합니다.
	var fileRecord models.File
	if err := fs.db.First(&fileRecord, "id = ?", fileID).Error; err != nil {
		return fmt.Errorf("failed to find file record: %w", err)
	}

	// S3에 저장된 객체의 키는 파일 업로드 시 사용한 규칙("files/{fileID}")를 따릅니다.
	s3Key := fmt.Sprintf("files/%s/%s", itemID, fileID.String())

	// S3에서 객체를 삭제합니다.
	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(fs.bucketName),
		Key:    aws.String(s3Key),
	}
	_, err := fs.s3Client.DeleteObject(ctx, deleteInput)
	if err != nil {
		return fmt.Errorf("failed to delete file from s3: %w", err)
	}

	// DB에서 파일 레코드를 삭제합니다.
	if err := fs.db.Delete(&fileRecord).Error; err != nil {
		return fmt.Errorf("failed to delete file record from db: %w", err)
	}

	return nil
}
