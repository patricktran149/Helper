package s3

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"os"
	"path/filepath"
)

type S3Custom struct {
	Host       string
	AccessKey  string
	SecretKey  string
	Region     string
	BucketName string
	S3         *s3.S3
}

func (s *S3Custom) NewConnection() (err error) {
	// Set up AWS session with custom endpoint
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(s.Region),
		Endpoint:    aws.String(s.Host),
		Credentials: credentials.NewStaticCredentials(s.AccessKey, s.SecretKey, ""),
	})
	if err != nil {
		err = errors.New("Failed to create session: " + err.Error())
	}

	// Create an S3 service client
	s.S3 = s3.New(sess)
	return nil
}

func (s *S3Custom) UploadFile(s3FileName, localFilePath string) error {
	file, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = s.S3.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(s3FileName),
		ACL:    aws.String("public-read"),
		Body:   file,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *S3Custom) UploadData(s3FileName string, data string) error {
	// Create a new file with the localFilePath and write data to it
	fileName := filepath.Base(s3FileName)
	localFilePath := fmt.Sprintf("./temporary/%s", fileName)
	err := os.WriteFile(localFilePath, []byte(data), 0644)
	if err != nil {
		return err
	}

	file, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer os.Remove(localFilePath)
	defer file.Close()

	_, err = s.S3.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(s3FileName),
		Body:   file,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *S3Custom) DownloadFile(s3FileName, localFilePath string) error {
	resp, err := s.S3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(s3FileName),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(localFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (s *S3Custom) DeleteFile(s3FileName string) error {
	// Delete the object
	_, err := s.S3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(s3FileName),
	})
	if err != nil {
		return errors.New("Failed to delete object: " + err.Error())
	}

	return nil
}
