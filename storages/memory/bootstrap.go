package memory

import (
	"broadcaster/storages"
	"context"
)

// Initializes DB from config file.
func (s *Storage) BootstrapFromConfigFile(ctx context.Context, uri string) error {
	feeds, err := storages.GetFeedsFromConfig(ctx, uri, s.logger)
	if err != nil {
		return err
	}
	s.logger.Debugf("Found '%d' feeds in config file", len(feeds))

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, feed := range feeds {
		conv := feed.ToRssFeed()

		if _, ok := s.feeds[conv.Id]; ok {
			s.logger.Debugf("Feed '%s' already in state", conv.Id)
			continue
		}

		s.logger.With("feed_id", conv.Id).Debug("Adding feed to state")

		s.feeds[conv.Id] = conv
	}

	s.logger.Infof("Loaded '%d' feeds", len(s.feeds))

	return nil
}
