package main

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func createS3Client(accessKeyID, secretAccessKey, region, endpointURL string) *s3.Client {
	return s3.New(s3.Options{
		Credentials:  credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		Region:       region,
		BaseEndpoint: &endpointURL,
		EndpointOptions: s3.EndpointResolverOptions{
			DisableHTTPS: true,
		},
		UsePathStyle: true,
	})
}

func createHttpRouter(s3Client *s3.Client) *gin.Engine {
	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Hello World!"})
	})

	router.GET("/download/:key", func(c *gin.Context) {
		log.Printf("Downloading %s", c.Param("key"))
		key := c.Param("key")

		output, err := s3Client.GetObject(c.Request.Context(), &s3.GetObjectInput{
			Bucket: aws.String("default"),
			Key:    aws.String(key),
		})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.DataFromReader(200, *output.ContentLength, *output.ContentType, output.Body, nil)
	})

	router.POST("/upload", func(c *gin.Context) {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		defer file.Close()

		key := uuid.New()

		_, err = s3Client.PutObject(c.Request.Context(), &s3.PutObjectInput{
			Bucket:      aws.String("default"),
			Key:         aws.String(key.String()),
			Body:        file,
			ContentType: &fileHeader.Header["Content-Type"][0],
		})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"key": key.String()})
	})

	return router
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")
	endpointURL := os.Getenv("AWS_ENDPOINT_URL")

	if accessKeyID == "" || secretAccessKey == "" || region == "" || endpointURL == "" {
		log.Fatal("AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION and AWS_ENDPOINT_URL must be set")
	}

	s3 := createS3Client(accessKeyID, secretAccessKey, region, endpointURL)
	router := createHttpRouter(s3)

	router.Run(":8080")
}
