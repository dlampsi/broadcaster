# Broadcaster

[![Image](https://github.com/dlampsi/broadcaster/actions/workflows/image.yml/badge.svg)](https://github.com/dlampsi/broadcaster/actions/workflows/image.yml)

A service that processes RSS feed items (optionally translates) and sends them to customized destinations (eg Slack or Telegam).

## Usage

```bash
# Show help
$ broadcaster --help

# Run application as a service
$ broadcaster server
```

## Configuration

### Environment variables

Available environment variables for service run type:

| Name | Description | Default value |
| ---- | ----------- | ------------- |
| `BCTR_ENV` | Environment name / id. | `local` |
| `BCTR_LOG_LEVEL` | Logging level. | `info` |
| `BCTR_LOG_FORMAT` | Logging format. Options: `json`, `pretty`, `pretty_color`, `do_app` | `pretty_color` |
| `BCTR_BOOTSTRAP_FILE` | Bootstrap config file path in uri format.<br>Example: `file:///path/to/config.yml` | |
| `BCTR_TRANSLATOR_TYPE` | Translation service type. * | `google_cloud` |
| `BCTR_GOOGLE_CLOUD_PROJECT_ID` | Google Cloud Project ID. | |
| `BCTR_GOOGLE_CLOUD_CREDS` | Google Cloud [application credentials](https://cloud.google.com/docs/authentication/provide-credentials-adc) string. Basically it should be conten of the credentials json file. <br> You can use `GOOGLE_APPLICATION_CREDENTIALS` env to specify path to credentials file.| |
| `BCTR_CHECK_INTERVAL` | Feeds fetch interval in seconds | `300` |
| `BCTR_BACKFILL_HOURS` | How many hours back to process feeds items. | `0` |
| `BCTR_MUTE_NOTIFICATIONS` | Disable sent notification to destinations. | `false` |
| `BCTR_TELEGRAM_BOT_TOKEN` | Telegram bot token. |  |
| `BCTR_SLACK_API_TOKEN` | Slack bot API token. |  |
| `BCTR_STATE_TTL` | Application state TTL in seconds. | `86400` (24h) |

\* Google Cloud Translation service requires `BCTR_GOOGLE_CLOUD_CREDS` or `GOOGLE_APPLICATION_CREDENTIALS` environment variable to be set.

### Bootstrap

Initial bootstrap configuration can be provided via `BCTR_BOOTSTRAP_FILE` environment variable. In that case service will upload specified feeds configurations from the provided config file.

Supported bootstap config sources and formats:

```bash
# Local filesystem
BCTR_BOOTSTRAP_FILE="file:///path/to/config.yml"
# Google Cloud Storage
BCTR_BOOTSTRAP_FILE="gs://bucket/path/to/config.yml"
# DigitalOcean Spaces *
BCTR_BOOTSTRAP_FILE="do://bucket/path/to/config.yml"
```

Config file format and example:

```yaml
feeds:
  - source: Dummy website
    category: Latest
    url: https://dummyfeed.com/rss
    language: fi
    translations:
      - to: en
    notifications:
      - type: slack
        to: ["#general"]
      - type: telegram
        to: ["-1234567890","-1234567891"]
```

## Notifiers

Broadcaster supports the following notifiers: `slack`, `telegram`.

To enable a notifier you should specify [corresponding](#environment-variables) TOKEN variable and add config to the `notify` section of the feed configuration.
