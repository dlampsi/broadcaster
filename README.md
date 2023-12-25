# Broadcaster

[![Image](https://github.com/dlampsi/broadcaster/actions/workflows/image.yml/badge.svg)](https://github.com/dlampsi/broadcaster/actions/workflows/image.yml)

A service that processes RSS feed items (optionally translates) and sends them to customized destinations (eg Telegam).

## Usage

```bash
# Show help
$ broadcaster --help

# Run application as a service
$ broadcaster server
```

## Environment variables

Available environment variables for service run type:

| Name | Description | Default value |
| ---- | ----------- | ------------- |
| `BCTR_ENV` | Environment name / id. | `local` |
| `BCTR_LOG_LEVEL` | Logging level. | `info` |
| `BCTR_LOG_FORMAT` | Logging format. Options: `json`, `pretty`, `pretty_color` | `json` |
| `BCTR_CONFIG` | Config file path in uri format.<br>Example: `file:///path/to/config.yml` | |
| `BCTR_TRANSLATOR_TYPE` | Translation service type. * | `google_cloud` |
| `BCTR_GOOGLE_CLOUD_PROJECT_ID` | Google Cloud Project ID. | |
| `BCTR_CHECK_INTERVAL` | Feeds fetch interval in seconds | `300` |
| `BCTR_BACKFILL_HOURS` | How many hours back to process feeds items. | `0` |
| `BCTR_MUTE_NOTIFICATIONS` | Disable sent notification to destinations. | `false` |
| `BCTR_TELEGRAM_BOT_TOKEN` | Telegram bot token. |  |
| `BCTR_SLACK_API_TOKEN` | Slack bot API token. |  |

\* Google Cloud Translation service requires `GOOGLE_APPLICATION_CREDENTIALS` environment variable to be set.

## Feeds configuration

Application uses config config file as a feeds configuration source. File can be loaded from local filesystem or Google Cloud Storage.

File path should be specified in the URL format, eg `file:///path/to/config.yml` or `gs://bucket/path/to/config.yml`. <br>
If config file is not specified, application will try to use `config.yml` file in the current directory.

Config file is a YAML file with the following structure:

```yaml
feeds:
  - source: News portal
    category: Latest
    url: https://dummyfeed.com/rss
    language: fi
    notify:
      - type: telegram
        to: ["-1234567890","-1234567891"]
      - type: slack
        to: ["#general"]
    translates:
      - to: en
```

## Notifiers

Broadcaster supports the following notifiers: `telegram`, `slack`.

To enable a notifier you should specify [corresponding](#environment-variables) TOKEN variable and add config to the `notify` section of the feed configuration.
