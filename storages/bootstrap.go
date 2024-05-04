package storages

import (
	"broadcaster/structs"
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
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type FeedConfig struct {
	Source        string                    `yaml:"source"`
	Category      string                    `yaml:"category"`
	URL           string                    `yaml:"url"`
	Language      string                    `yaml:"language"`
	ItemsLimit    int                       `yaml:"items_limit"`
	Disabled      bool                      `yaml:"disabled"`
	Notifications []FeedNotificationsConfig `yaml:"notifications"`
}

type FeedNotificationsConfig struct {
	Type      string                 `yaml:"type"`
	To        []string               `yaml:"to"`
	Muted     bool                   `yaml:"muted"`
	Translate FeedTranslationsConfig `yaml:"translate"`
}

type FeedTranslationsConfig struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

func (c FeedConfig) ToRssFeed() structs.RssFeed {
	feedid := strings.ReplaceAll(c.Source, " ", "")
	if c.Category != "" {
		feedid += "." + strings.ReplaceAll(c.Category, " ", "")
	}

	result := structs.RssFeed{
		Id:         feedid,
		Source:     c.Source,
		Category:   c.Category,
		URL:        c.URL,
		Language:   c.Language,
		ItemsLimit: c.ItemsLimit,
	}

	for _, n := range c.Notifications {
		rn := structs.RssFeedNotification{
			Type:  n.Type,
			To:    n.To,
			Muted: n.Muted,
			Translate: structs.RssFeedTranslation{
				From: coalesce(n.Translate.From, c.Language),
				To:   n.Translate.To,
			},
		}
		result.Notifications = append(result.Notifications, rn)
	}

	return result
}

// Loads feeds configurations from file.
func GetFeedsFromConfig(ctx context.Context, uri string, logger *zap.SugaredLogger) ([]FeedConfig, error) {
	data, err := loadFileByUri(ctx, uri, logger)
	if err != nil {
		return nil, fmt.Errorf("Failed to load by uri: %w", err)
	}

	var config struct {
		Feeds []FeedConfig `yaml:"feeds"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("Failed to parse data: %w", err)
	}

	return config.Feeds, nil
}

func loadFileByUri(ctx context.Context, uri string, logger *zap.SugaredLogger) ([]byte, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse config uri: %w", err)
	}

	switch parsed.Scheme {
	case "file":
		logger.With("path", parsed.Path).Debug("Loading from filesytsem")
		return os.ReadFile(parsed.Path)

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

		return io.ReadAll(rc)

	case "do":
		parsed.Path = strings.TrimLeft(parsed.Path, "/")

		logger.With("bucket", parsed.Host, "object", parsed.Path).
			Debug("Loading config file from Digital Ocean Spaces")

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

		return io.ReadAll(obj.Body)

	default:
		return nil, errors.New("Unknown config file URL scheme")
	}
}

// Returns the first non-empty string from the given list of strings.
// If all strings are empty, returns an empty string.
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
