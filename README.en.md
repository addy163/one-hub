<p align="right">
   <strong>English</strong> | <a href="./README.md">中文</a>
</p>

<p align="center">
   <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://github.com/MartialBE/one-hub/assets/42402987/c4125d1a-5577-446d-ba15-2a71c52140c1">
   <img height="90" src="https://raw.githubusercontent.com/MartialBE/one-api/main/web/src/assets/images/logo.svg">
   </picture>
</p>

<div align="center">

# One Hub

_This project is a fork of [MartialBE/one-hub](https://github.com/MartialBE/one-hub), which itself is a fork of [one-api](https://github.com/songquanpeng/one-api)._

<p align="center">
  <a href="https://raw.githubusercontent.com/MartialBE/one-api/main/LICENSE">
    <img src="https://img.shields.io/github/license/MartialBE/one-api?color=brightgreen" alt="license">
  </a>
  <a href="https://github.com/MartialBE/one-hub/releases/latest">
    <img src="https://img.shields.io/github/v/release/MartialBE/one-api?color=brightgreen&include_prereleases" alt="release">
  </a>
  <a href="https://github.com/users/MartialBE/packages/container/package/one-api">
    <img src="https://img.shields.io/badge/docker-ghcr.io-blue" alt="docker">
  </a>
  <a href="https://hub.docker.com/r/martialbe/one-api">
    <img src="https://img.shields.io/badge/docker-dockerHub-blue" alt="docker">
  </a>
  <a href="https://goreportcard.com/report/github.com/MartialBE/one-api">
    <img src="https://goreportcard.com/badge/github.com/MartialBE/one-api" alt="GoReportCard">
  </a>
</p>

**Please do not mix with the original version, because the new functions, the database is not compatible with the original version**

**For the sake of simplicity, after this project, except for updating the model list built into the program when adding a new supplier, the model list built into the program will not be updated under normal circumstances.**

If you find that a new model is missing, please update the newly added model in `Backend-Model Price-Update Price`

[Demo Site](https://one-hub.xiao5.info/)
[Documentation](https://one-hub-doc.vercel.app/)

</div>

## About this repository

This repository is a **secondary development of [one-hub](https://github.com/MartialBE/one-hub)** (one-hub is a secondary development of [one-api](https://github.com/songquanpeng/one-api)).

- Repository: <https://github.com/addy163/one-hub>
- Upstream: <https://github.com/MartialBE/one-hub>

All upstream features and the data schema are preserved; changes are incremental only:

| Area | Change |
| --- | --- |
| Multi-endpoint distribution | Admins maintain a list of API base URLs and expose different ones per user group (`vip` → main site, `default` → alternate domain). Filtering happens server-side, `groups` is never sent to clients, and the frontend falls back to `ServerAddress` when no endpoint is configured |
| Endpoint health check | The server periodically probes each endpoint's `/api/status`, exposes the result through `/api/status`, and the admin grid shows a status column plus a "probe now" button. Interval/timeout are configurable (`RelayEndpointProbeInterval`, `RelayEndpointProbeTimeout`) |
| Pricing UI fix | A model that is not available for a group is now rendered as grey `N/A` instead of `Free`, so "no permission" is no longer confused with "free" |
| Passthrough semantics | `ShouldPassthroughExtraBody` makes the three layers explicit: per-channel `disable_extra_body` is a veto, the global switch is the master switch, `allow_extra_body` is a per-channel whitelist when the global switch is off |
| Security / reverse proxy | `session_secure: auto` honours `X-Forwarded-Proto/Scheme` and `Forwarded`; `trusted_proxies` restricts which proxies may set `X-Forwarded-For`; WebSocket connection limits are configurable |
| Built-in data | Placeholder default prices and 108 model specs (including Doubao Seed 2.1 Lite and Seed Character) are shipped with the binary |
| Time-based rates | A model can define several time windows (e.g. 22:00–08:00 at half price). Windows are evaluated in UTC+8; requests hitting a window are billed with that window's ratios, and the log records `price_period` |

## Database changes and automatic migration

No manual SQL is required, for either a fresh install or an upgrade.

- **Schema**: GORM `AutoMigrate` runs on every start (`model/main.go`), creating missing tables, columns and indexes.
- **Data**: `gormigrate` migrations in `model/migrate.go`, recorded in the `migrations` table with unique IDs. Each migration is idempotent and only fills in missing rows, never overwriting values an admin already changed. They run on the master node only.
- **Current fork migrations**: `202611170001` seeds the three relay endpoint options, any missing placeholder price row and the model specs from `model/seed/model_info.json`; `202611170002` re-runs the same idempotent seeding so that databases already upgraded past `202611170001` still receive later additions to the seed data.
- **Built-in prices**: on a fresh install with an empty `prices` table, `GetDefaultPrice()` seeds the built-in prices. Those values are placeholders only — provider pricing varies, so adjust them under "Model Prices"; rows you edit are never overwritten by later upgrades. The migration intentionally skips the price row when the table is empty so that the built-in sync is not blocked.

### Time-based rates (peak / off-peak)

Some providers (Alibaba Model Studio, Volcengine Ark) charge different rates inside and outside peak hours. Any model can carry several time windows; billing picks the window that contains the request arrival time.

Configure it under "Pricing → Single price" by enabling "Time-based rates" and entering CNY per 1M tokens per window (converted to ratios on save).

```json
{
  "enabled": true,
  "periods": [
    { "name": "off-peak", "start": "22:00", "end": "08:00", "input": 0.07143, "output": 0.28571, "cached": 0.02 }
  ]
}
```

Rules: windows use **UTC+8** regardless of the server time zone; the start is inclusive and the end exclusive; `start > end` means the window crosses midnight; windows are matched in order and the first hit wins; outside every window the base `prices.input/output` applies; `input`/`output` of 0 inherits the base rate; `cached` overrides the cache-hit ratio for that window. Pre-consumption and settlement use the same window (a long request is not re-priced mid-flight), and the log `metadata` records `price_period` together with the effective `input_ratio` / `output_ratio`.

With no windows configured, `time_prices` stays empty and behaviour is identical to upstream.

### Migrating from one-hub

```bash
mysqldump -h127.0.0.1 -u<user> -p<pass> onehub > onehub_before_fork.sql
docker stop one-hub && docker rename one-hub one-hub-prev
docker run -d --name one-hub --network host -v /path/to/data:/data ghcr.io/addy163/one-hub:latest
```

Watch the startup log for `database migrated` and the `已补充...` lines, then verify that the "Relay endpoints" panel exists, "Model Prices" opens with the built-in prices initialized, and the model info table contains 108 rows. To roll back, restore the old container and the SQL dump.

## Functional Changes


- Brand new UI interface
- Added user dashboard
- Added administrator data analysis and statistics interface
- Refactored the intermediary `supplier` module
- Support for using `Azure Speech` to simulate `TTS` function
- Channels can be configured with separate http/socks5 proxies
- Support for dynamically returning user model lists
- Support for custom speed testing models
- Logs now include request duration
- Support and optimize function calls for non-OpenAI models (supported models can be used directly in Lobe-Chat)
- Support for custom completion rates
- Support for full pagination and sorting
- Support for `Telegram bot`
- Support for models charged per use
- Support for model wildcards
- Support for starting the program using a configuration file
- Support model price update
- Support automatic acquisition of supplier models
- Support chat only. If there is an incoming `function call` parameter after opening, the channel will be skipped.
- Support payment
- Support configuring user group RPM
- Support `Prometheus` monitoring

## Current Supported Providers

| Provider                                                              | Chat                             | Embeddings | Audio  | Images   | Other                                                            |
| --------------------------------------------------------------------- | -------------------------------- | ---------- | ------ | -------- | ---------------------------------------------------------------- |
| [OpenAI](https://platform.openai.com/docs/api-reference/introduction) | ✅                               | ✅         | ✅     | ✅       | -                                                                |
| [Azure OpenAI](https://oai.azure.com/)                                | ✅                               | ✅         | ✅     | ✅       | -                                                                |
| [Azure Speech](https://portal.azure.com/)                             | -                                | -          | ⚠️ tts | -        | -                                                                |
| [Anthropic](https://www.anthropic.com/)                               | ✅                               | -          | -      | -        | -                                                                |
| [Gemini](https://aistudio.google.com/)                                | ✅                               | -          | -      | -        | -                                                                |
| [百度文心](https://console.bce.baidu.com/qianfan/overview)            | ✅                               | ✅         | -      | -        | -                                                                |
| [通义千问](https://dashscope.console.aliyun.com/overview)             | ✅                               | ✅         | -      | -        | -                                                                |
| [讯飞星火](https://console.xfyun.cn/)                                 | ✅                               | -          | -      | -        | -                                                                |
| [智谱](https://open.bigmodel.cn/overview)                             | ✅                               | ✅         | -      | ⚠️ image | -                                                                |
| [腾讯混元](https://cloud.tencent.com/product/hunyuan)                 | ✅                               | -          | -      | -        | -                                                                |
| [百川](https://platform.baichuan-ai.com/console/apikey)               | ✅                               | ✅         | -      | -        | -                                                                |
| [MiniMax](https://www.minimaxi.com/user-center/basic-information)     | ✅                               | ✅         | -      | -        | -                                                                |
| [Deepseek](https://platform.deepseek.com/usage)                       | ✅                               | -          | -      | -        | -                                                                |
| [Moonshot](https://moonshot.ai/)                                      | ✅                               | -          | -      | -        | -                                                                |
| [Mistral](https://mistral.ai/)                                        | ✅                               | ✅         | -      | -        | -                                                                |
| [Groq](https://console.groq.com/keys)                                 | ✅                               | -          | -      | -        | -                                                                |
| [Amazon Bedrock](https://console.aws.amazon.com/bedrock/home)         | ⚠️ Only support Anthropic models | -          | -      | -        | -                                                                |
| [零一万物](https://platform.lingyiwanwu.com/details)                  | ✅                               | -          | -      | -        | -                                                                |
| [Cloudflare AI](https://ai.cloudflare.com/)                           | ✅                               | -          | ⚠️ stt | ⚠️ image | -                                                                |
| [Midjourney](https://www.midjourney.com/)                             | -                                | -          | -      | -        | [midjourney-proxy](https://github.com/novicezk/midjourney-proxy) |
| [Cohere](https://cohere.com/)                                         | ✅                               | -          | -      | -        | -                                                                |
| [Stability AI](https://platform.stability.ai/account/credits)         | -                                | -          | -      | ⚠️ image | -                                                                |
| [Coze](https://www.coze.com/open/docs/chat?_lang=zh)                  | ✅                               | -          | -      | -        | -                                                                |
| [Ollama](https://github.com/ollama/ollama)                            | ✅                               | ✅         | -      | -        | -                                                                |
| [Suno](https://suno.com/)                                             | -                                | -          | -      | -        | [Suno-API](https://github.com/Suno-API/Suno-API)                 |

## Acknowledgements

- This program utilizes the following open-source projects:
  - [one-api](https://github.com/songquanpeng/one-api) serves as the foundation of this project.
  - [Berry Free React Admin Template](https://github.com/codedthemes/berry-free-react-admin-template) provides the frontend interface for this project.
  - [minimal-ui-kit](https://github.com/minimal-ui-kit/material-kit-react), some styles from this project were used.
  - [new api](https://github.com/Calcium-Ion/new-api), the code for the Midjourney/Suno module is sourced from here.

Special thanks to the authors and contributors of the above projects.

## Others

<a href="https://next.ossinsight.io/widgets/official/analyze-repo-stars-history?repo_id=689214770" target="_blank" style="display: block" align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://next.ossinsight.io/widgets/official/analyze-repo-stars-history/thumbnail.png?repo_id=689214770&image_size=auto&color_scheme=dark" width="721" height="auto">
    <img alt="Star History of MartialBE/one-api" src="https://next.ossinsight.io/widgets/official/analyze-repo-stars-history/thumbnail.png?repo_id=689214770&image_size=auto&color_scheme=light" width="721" height="auto">
  </picture>
</a>
