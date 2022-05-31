package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Config struct {
	awsAccessKeyId     string
	awsSecretAccessKey string
	awsDefaultRegion   string
	awsEndpointUrl     string
	awsNoVerifySsl     bool
}

// Validate just validates that the config is not empty
func (config *S3Config) Validate() error {
	if awsAccessKeyId == "" {
		return fmt.Errorf("cannot get awsAccessKeyId")
	}

	if awsSecretAccessKey == "" {
		return fmt.Errorf("cannot get awsSecretAccessKey")
	}

	if awsDefaultRegion == "" {
		return fmt.Errorf("cannot get awsDefaultRegion")
	}

	if awsEndpointUrl != "" {
		_, err := url.Parse(awsEndpointUrl)
		if err != nil {
			return fmt.Errorf("Invalid aweEndpointURL %s: %v", awsEndpointUrl, err)
		}
	}

	return nil
}

// GetAwsSession returns aws session with the given details
func (config *S3Config) GetAwsSession() (*session.Session, error) {
	sess, err := session.NewSession(
		&aws.Config{
			Region:           aws.String(config.awsDefaultRegion),
			S3ForcePathStyle: aws.Bool(true),
			Credentials: credentials.NewStaticCredentials(
				config.awsAccessKeyId,
				config.awsSecretAccessKey,
				"",
			),
		})
	if err != nil {
		return nil, err
	}

	if config.awsEndpointUrl != "" {
		sess.Config.Endpoint = aws.String(config.awsEndpointUrl)
	}

	if config.awsNoVerifySsl {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402
		sess.Config.HTTPClient = &http.Client{
			Transport: transport,
		}
	}

	return sess, nil
}

func (config *S3Config) UploadFile(sess *session.Session, targetOutput string, filenamePath string) error {

	u, _ := url.Parse(targetOutput)
	if u.Scheme != "s3" {
		return fmt.Errorf("not valid target '%s'", u.Scheme)
	}

	fp, err := os.Open(filenamePath)
	if err != nil {
		return fmt.Errorf("not valid source file: %v", err)
	}
	uploader := s3manager.NewUploader(sess)
	// fmt.Printl("test/eloy.iso")
	up, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(u.Host),
		Key:    aws.String(u.Path),
		Body:   fp,
	})

	if err != nil {
		return fmt.Errorf("cannot upload file correctly: %v", err)
	}

	InfoLogger.Printf(
		"Uploaded correctly the file to '%s' with uploadID='%s'",
		targetOutput, up.UploadID)
	return err
}
