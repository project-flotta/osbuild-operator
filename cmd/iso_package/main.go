package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2" // imports as package "cli"
)

const (
	armArchValue = "aarch"
	x86ArchValue = "x86"

	awsAccessKeyId     = "aws-access-key-id"
	awsSecretAccessKey = "aws-secret-access-key" //nolint:gosec
	awsDefaultRegion   = "aws-default-region"
	awsEndpointUrl     = "aws-endpoint-url"
	awsNoVerifySsl     = "aws-no-verify-ssl"
)

var (
	app *cli.App

	InfoLogger    = log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime)
	WarningLogger = log.New(os.Stderr, "WARNING: ", log.Ldate|log.Ltime)
	ErrorLogger   = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime)

	allowedArchs = []string{x86ArchValue, armArchValue}
)

func init() {
	app = cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "source",
			Aliases:  []string{"s"},
			Usage:    "ISO source url to use as based image",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "kickstart",
			Aliases:  []string{"ks"},
			Usage:    "Kickstart file to append to the iso",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "arch",
			Usage: "Arch to compile to, possible values: x86, aarch. Default x86",
			Value: "x86",
		},
		&cli.StringFlag{
			Name:  "upload-target",
			Usage: "target to upload the s3 file to, like s3://mybucket/myfile.iso",
		},
		&cli.StringFlag{
			Name:    awsAccessKeyId,
			Usage:   "AWS_ACCESS_KEY_ID to store the iso on s3",
			EnvVars: []string{"AWS_ACCESS_KEY_ID"},
		},
		&cli.StringFlag{
			Name:    awsSecretAccessKey,
			Usage:   "AWS_SECRET_ACCESS_KEY to store the iso on s3",
			EnvVars: []string{"AWS_SECRET_ACCESS_KEY"},
		},
		&cli.StringFlag{
			Name:    awsDefaultRegion,
			Usage:   "AWS_DEFAULT_REGION to store the iso on s3",
			EnvVars: []string{"AWS_DEFAULT_REGION"},
		},
		&cli.StringFlag{
			Name:    awsEndpointUrl,
			Usage:   "AWS_ENDPOINT_URL to use when uploading the iso image",
			EnvVars: []string{"AWS_ENDPOINT_URL"},
		},
		&cli.BoolFlag{
			Name:    awsNoVerifySsl,
			Usage:   "AWS_NO_VERIFY_SSL to not validate ssl connection",
			EnvVars: []string{"AWS_NO_VERIFY_SSL"},
			Value:   false,
		},
	}
}

func main() {
	app.Action = func(c *cli.Context) error {
		arch := c.String("arch")
		archMatched := false
		for _, possibleArch := range allowedArchs {
			if arch == possibleArch {
				archMatched = true
				break
			}
		}

		if !archMatched {
			log.Fatalf("Arch value '%s' is not allowed. Allowed arches are: %v", arch, allowedArchs)
		}

		build := Builder{
			Source:    c.String("source"),
			Kickstart: c.String("kickstart"),
			Arch:      arch,
		}

		_, err := build.Run()
		if err != nil {
			return err
		}
		targetUpload := c.String("upload-target")
		if targetUpload != "" {
			creds := &S3Config{
				awsAccessKeyId:     c.String(awsAccessKeyId),
				awsSecretAccessKey: c.String(awsSecretAccessKey),
				awsDefaultRegion:   c.String(awsDefaultRegion),
				awsEndpointUrl:     c.String(awsEndpointUrl),
				awsNoVerifySsl:     c.Bool(awsNoVerifySsl),
			}

			err := creds.Validate()
			if err != nil {
				return fmt.Errorf("cannot push information to s3: %v", err)
			}

			return build.Upload(creds, targetUpload)
		}
		return err
	}

	err := app.Run(os.Args)
	if err != nil {
		ErrorLogger.Fatal(err)
	}

}
