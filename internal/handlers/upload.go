package handlers

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	appConfig "github.com/pushp314/devconnect-backend/internal/config"
	"github.com/pushp314/devconnect-backend/pkg/utils"
)

// -- Helpers -- //

func getS3Client() (*s3.Client, error) {
	cfg := appConfig.AppConfig
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID),
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.R2AccessKeyID, cfg.R2SecretAccessKey, "")),
	)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(awsCfg), nil
}

// -- Handlers -- //

func UploadFile(c *gin.Context) {
	// 1. Get File
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		// Fallback fields logic
		file, header, err = c.Request.FormFile("image")
		if err != nil {
			file, header, err = c.Request.FormFile("media")
			if err != nil {
				file, header, err = c.Request.FormFile("video")
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "No valid file field found"})
					return
				}
			}
		}
	}
	defer file.Close()

	// 2. Generate Key
	ext := filepath.Ext(header.Filename)
	key := fmt.Sprintf("%s/%s%s", c.DefaultQuery("folder", "uploads"), utils.GenerateID(), ext)

	// 3. Upload to R2
	client, err := getS3Client()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to init storage client"})
		return
	}

	cfg := appConfig.AppConfig
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.R2BucketName),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(header.Header.Get("Content-Type")),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed: " + err.Error()})
		return
	}

	// 4. Return URL
	// Public URL construction depends on R2 setup (Custom Domain or R2.dev)
	publicURL := cfg.R2PublicURL
	if publicURL == "" {
		// Fallback or warning
		publicURL = fmt.Sprintf("https://%s.r2.dev", cfg.R2BucketName) // Simplified guess
	}

	fullURL := fmt.Sprintf("%s/%s", publicURL, key)

	c.JSON(http.StatusOK, gin.H{
		"url":      fullURL,
		"key":      key,
		"mimetype": header.Header.Get("Content-Type"),
		"size":     header.Size,
	})
}

// Wrappers
func UploadProfileImage(c *gin.Context) {
	c.Request.URL.RawQuery = "folder=devconnect/profiles"
	UploadFile(c)
}

func UploadStoryMedia(c *gin.Context) {
	c.Request.URL.RawQuery = "folder=devconnect/stories"
	UploadFile(c)
}

func UploadReelVideo(c *gin.Context) {
	c.Request.URL.RawQuery = "folder=devconnect/reels"
	UploadFile(c)
}

func UploadChatAttachment(c *gin.Context) {
	c.Request.URL.RawQuery = "folder=devconnect/chat"
	UploadFile(c)
}
