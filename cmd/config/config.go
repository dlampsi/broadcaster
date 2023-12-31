package config

import (
	"broadcaster/structs"
	"broadcaster/utils/logging"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	googleStorage "cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsCreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gopkg.in/yaml.v3"
)

var (
	UnknownSchemeError error = errors.New("Unknown config file URL scheme")
)

type ConfigFile struct {
	Feeds []structs.FeedConfig `yaml:"feeds"`
}

func Load(ctx context.Context, uri string) (*ConfigFile, error) {
	logger := logging.FromContext(ctx)
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse config uri: %w", err)
	}

	var result ConfigFile

	var data []byte

	switch parsed.Scheme {
	case "file":
		logger.With("path", parsed.Path).Debug("Loading from filesytsem")

		cfile, err := os.ReadFile(parsed.Path)
		if err != nil {
			return nil, fmt.Errorf("Failed to load file: %w", err)
		}
		data = cfile

	case "gs":
		logger.With("bucket", parsed.Host, "object", parsed.Path).Debug("Loading from Google Cloud Storage")

		cl, err := googleStorage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("Failed to create google storage client: %w", err)
		}
		defer cl.Close()

		bucket := cl.Bucket(parsed.Host)

		gsPath := strings.TrimLeft(parsed.Path, "/")
		obj := bucket.Object(gsPath)

		rc, err := obj.NewReader(ctx)
		if err != nil {
			return nil, fmt.Errorf("Failed to read bucket object %s: %w", gsPath, err)
		}
		defer rc.Close()

		body, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("Failed to read object from reader: %w", err)
		}
		data = body

	case "do":
		parsed.Path = strings.TrimLeft(parsed.Path, "/")

		dlogger := logger.With("bucket", parsed.Host, "object", parsed.Path)
		dlogger.Debug("Loading config file from Digital Ocean Spaces")

		key := os.Getenv("DO_SPACES_ACCESS_KEY_ID")
		secret := os.Getenv("DO_SPACES_SECRET_ACCESS_KEY")
		creds := awsCreds.NewStaticCredentialsProvider(key, secret, "")

		resolver := aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				doRegion := os.Getenv("DO_SPACES_REGION")
				return aws.Endpoint{
					URL: fmt.Sprintf("https://%s.digitaloceanspaces.com", doRegion),
				}, nil
			},
		)

		cfg, err := awsConfig.LoadDefaultConfig(ctx,
			awsConfig.WithRegion("us-east-1"),
			awsConfig.WithCredentialsProvider(creds),
			awsConfig.WithEndpointResolverWithOptions(resolver),
		)
		if err != nil {
			return nil, fmt.Errorf("Failed to load aws config: %w", err)
		}

		cl := s3.NewFromConfig(cfg)

		input := &s3.GetObjectInput{
			Bucket: aws.String(parsed.Host),
			Key:    aws.String(parsed.Path),
		}
		obj, err := cl.GetObject(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("Failed to get object: %w", err)
		}
		defer obj.Body.Close()

		body, err := io.ReadAll(obj.Body)
		if err != nil {
			return nil, fmt.Errorf("Failed to read object from reader: %w", err)
		}
		data = body

	default:
		return nil, UnknownSchemeError
	}

	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("Failed to parse data: %w", err)
	}
	return &result, nil
}
