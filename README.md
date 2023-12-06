# a0feed

[![Build](https://github.com/dlampsi/a0feed/actions/workflows/build.yml/badge.svg)](https://github.com/dlampsi/a0feed/actions/workflows/build.yml)

A0 Feed is a news translation service or application that processes RSS feed items, translates and sends them to customized destinations (eg Telegam).

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
    translates:
      - from: fi
        to: en
        notify:
          - type: telegram
            chat_id: <telegram_chat_id>
```
