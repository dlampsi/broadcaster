package config

import (
	"a0feed/structs"
	"a0feed/utils/logging"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	googleStorage "cloud.google.com/go/storage"
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

	default:
		return nil, UnknownSchemeError
	}

	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("Failed to parse data: %w", err)
	}
	return &result, nil
}
