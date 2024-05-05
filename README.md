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

<!-- ------------------------------------------------------------------------------------------ -->
## Translation services

Broadcaster supports the following translation services:

### Google Translate API

Free to use. No additional settings are required. Used by default.

### Google Cloud Translate

You will need to create (or already have) a Google Cloud account and a Project to use the Translatoin API. Google Cloud has a free tier for the translation service.

<!-- ------------------------------------------------------------------------------------------ -->
## Notifications

At this moment, Broadcaster supports sending notifications to [Slack](https://slack.com/) and [Telegram](https://telegram.org/).

To enable a notifier you should specify [corresponding](#environment-variables) token env variables and add config to the `notify` section of the feed configuration.

## Configuration

### Environment variables

Available environment variables for service run type:

| Name | Description | Default value |
| ---- | ----------- | ------------- |
| `BCTR_ENV` | Environment name / id. Mostly used for observability attributes. | `local` |
| `BCTR_LOG_LEVEL` | Application logging level. | `info` |
| `BCTR_LOG_FORMAT` | Logging format. Options: `json`, `pretty`, `pretty_color`, `do_app` | `pretty_color` |
| `BCTR_BOOTSTRAP_FILE` | Bootstrap config file path in uri format. See more in [Bootstrap](#bootstrap). | |
| `BCTR_TRANSLATOR_TYPE` | Translation service type to use. Options: `google_api`, `google_cloud`  | `google_api` |
| `BCTR_CHECK_INTERVAL` | Feeds fetch interval in seconds. | `300` |
| `BCTR_BACKFILL_HOURS` | How many hours back to process feeds items. For debugging purposes. | `0` |
| `BCTR_STATE_TTL` | Application state TTL in seconds. | `86400` (24h) |
| `BCTR_MUTE_NOTIFICATIONS` | Disable sent notification to destinations. For debugging purposes. | `false` |
| `BCTR_TELEGRAM_BOT_TOKEN` | Telegram bot token.<br>To send notifications to Telegram, you will need to create a [bot](https://core.telegram.org/bots/tutorial) and such a token. |  |
| `BCTR_SLACK_API_TOKEN` | Slack bot API token.<br>To send notifications to Slack, you will need to create an [application](https://api.slack.com/start/quickstart) and such a token. |  |

#### Google Cloud Translation API

If you plan to use Google Cloud Translate, you will need the following env variables:

| Name | Description |
| ---- | ----------- |
| `BCTR_GOOGLE_CLOUD_PROJECT_ID` | Google Cloud Project ID. |
| `BCTR_GOOGLE_CLOUD_CREDS` | Google Cloud [application credentials](https://cloud.google.com/docs/authentication/provide-credentials-adc) string. Basically it should be conten of the credentials json file. <br> You can use `GOOGLE_APPLICATION_CREDENTIALS` env to specify path to credentials file.|

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
    notifications:
      - type: slack
        to: ["#general"]
      - type: telegram
        to: ["-1234567890","-1234567891"]
        translate:
          to: en
```
