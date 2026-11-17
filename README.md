<p align="right">
   <strong>中文</strong> | <a href="./README.en.md">English</a>
</p>

<p align="center">
   <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://github.com/MartialBE/one-hub/assets/42402987/c4125d1a-5577-446d-ba15-2a71c52140c1">
   <img height="90" src="https://raw.githubusercontent.com/MartialBE/one-api/main/web/src/assets/images/logo.svg">
   </picture>
</p>

<div align="center">

# One Hub

_本项目基于 [MartialBE/one-hub](https://github.com/MartialBE/one-hub) 二次开发（one-hub 本身基于 [one-api](https://github.com/songquanpeng/one-api)）_

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

**请不要和原版混用，因为新增功能，数据库与原版不兼容。本仓库在上游基础上又增加了配置项与迁移记录，回退到上游前请先备份数据库**

**为了更加简洁，本项目之后，除了新增供应商时会更新程序自带的模型列表，平常不再更新程序自带的模型列表。**

**如果发现缺少新模型，请在`后台-模型价格-更新价格`中更新新增的模型**

[演示网站](https://one-hub.xiao5.info/)
[文档](https://one-hub-doc.vercel.app/)

</div>

> [!WARNING]
> 本项目为个人学习使用，不保证稳定性，且不提供任何技术支持，使用者必须在遵循 OpenAI 的使用条款以及法律法规的情况下使用，不得用于非法用途。  
> 根据[《生成式人工智能服务管理暂行办法》](http://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm)的要求，请勿对中国地区公众提供一切未经备案的生成式人工智能服务。

## 关于这个仓库

本项目是 [MartialBE/one-hub](https://github.com/MartialBE/one-hub) 的二次开发版本。上游 one-hub 本身基于 [songquanpeng/one-api](https://github.com/songquanpeng/one-api) 二次开发，因此本项目是「one-api → one-hub → 本仓库」这条线上的第三层。

- 本仓库：<https://github.com/addy163/one-hub>
- 上游：<https://github.com/MartialBE/one-hub>

### 和上游的关系

- **保留上游全部功能与数据结构**，不删功能、不改表名，只做增量；
- **增量改动集中在四块**：多线路地址分发、线路健康检查、计费/价格展示修正、反代与会话安全加固；
- **数据库变更全部自动完成**：结构靠 GORM `AutoMigrate`，数据靠 `gormigrate` 版本化迁移，升级只需换镜像，不需要手工执行 SQL（详见[数据库变更与自动迁移](#数据库变更与自动迁移)）。

> [!IMPORTANT]
> 与上游 README 相同，「不要和原版混用」依然成立。本项目额外写入了 `RelayEndpoints` 等配置项与迁移记录，从本版本回退到上游版本前请先备份数据库。

## 二次开发更新清单

### 2026-11-17 · 线路健康检查与迁移自动化

| 变更 | 说明 | 影响 |
| --- | --- | --- |
| 线路健康检查 | 服务端按间隔探测每条线路的 `/api/status`，结果写入内存并在 `/api/status` 中随线路一起下发；后台线路表格显示状态列并支持「立即检测」 | 新增配置项 `RelayEndpointProbeInterval`（秒，0 为关闭）、`RelayEndpointProbeTimeout`（毫秒）；新增接口 `GET /api/relay_endpoints`、`POST /api/relay_endpoints/probe` |
| 探测失败信息隔离 | 探测失败的具体原因（内网地址、证书错误等）只返回给管理员，普通用户只看到「离线」 | 安全加固 |
| 新用户优选在线线路 | 用户没有历史选择时，默认选中第一条在线线路，而不是固定取列表第一条 | 体验优化 |
| 内置模型规格 | 随程序内置 104 条模型规格（上下文长度、最大输出、输入输出模态） | 全新安装开箱即有模型详情，不再是一片空白 |
| 数据迁移自动化 | 新增 `gormigrate` 迁移 `202611170001`：补齐线路配置项、缺失的价格行、模型规格 | 老库升级自动补齐，只补缺失、不覆盖已有配置 |

### 2026-11-16 · 多线路地址分发

| 变更 | 说明 |
| --- | --- |
| 多线路分发 | 后台「设置 → 运营设置 → 线路管理」维护地址清单，可按用户分组下发不同地址（如 `vip` 走主站、`default` 走备用域名） |
| 服务端分组过滤 | 线路按 `group` 在服务端裁剪，`groups` 字段不下发，避免用户从 F12 里看到其它分组的地址 |
| 前端跟随 | 令牌页、仪表盘快捷接入、Playground 三处统一跟随当前选中线路，选择记在 localStorage |
| 兼容回退 | 未配置任何线路时前端回落到 `ServerAddress`，行为与升级前完全一致 |
| 价格弹窗修正 | 价格明细里「该分组未开通此模型」不再显示成 `Free`，改为灰显 + `N/A`，避免把「没权限」误读成「免费」 |

### 2026-11-15 · 计费、安全与反代

| 变更 | 说明 |
| --- | --- |
| 透传三层语义固化 | 抽出 `ShouldPassthroughExtraBody(渠道否决, 渠道白名单)`：渠道 `disable_extra_body` 是否决票，全局开关是总闸，`allow_extra_body` 是全局关闭时的白名单，附带 7 个表驱动单测 |
| 会话安全 | `session_secure: auto` 按请求协议动态决定是否加 `Secure`，兼容 `X-Forwarded-Proto/Scheme`、`Forwarded` 等反代头；`session_secret` 为弱值时自动替换为随机值 |
| 可信代理 | 支持 `trusted_proxies`，只信任来自反代的 `X-Forwarded-For`，避免伪造 IP 绕过限流 |
| WebSocket 治理 | 新增 `ws_max_connections`、`ws_max_connections_per_token`、`ws_idle_timeout_seconds` 等上限配置 |
| 修复「恢复默认」 | 删除配置项时 `key` 是 MySQL 保留字，按方言加引号，修复 1064 错误 |

## 数据库变更与自动迁移

本项目所有数据库变更都随程序启动自动完成，**升级不需要手工执行 SQL**，也不需要额外的迁移脚本。

### 结构变更：GORM AutoMigrate

启动时依次对全部模型执行 `AutoMigrate`（`model/main.go`），缺表建表、缺列加列、缺索引补索引。新增字段只要写在结构体里，老库升级和全新安装都会自动对齐。

### 数据变更：gormigrate 版本化迁移

需要写入数据（默认配置项、内置价格、模型规格）的改动放在 `model/migrate.go` 的迁移列表里，由 `gormigrate` 记录到 `migrations` 表：

- 每个迁移有唯一 ID（如 `202611170001`），执行过就不会重复执行；
- 迁移内部按「只补缺失、不覆盖已有数据」实现，即使重复执行也安全；
- 全新安装与老库升级走同一套代码，不需要区分两条路径；
- 主从部署时只在主库执行（`IsMasterNode` 控制）。

当前与上游不同的迁移只有一个：

| ID | 内容 |
| --- | --- |
| `202611170001` | 补齐 `RelayEndpoints` / `RelayEndpointProbeInterval` / `RelayEndpointProbeTimeout` 三个配置项；补齐缺失的价格占位行（仅在该模型尚无价格时插入）；补齐 104 条模型规格（数据见 `model/seed/model_info.json`） |

### 内置价格表

全新安装且 `prices` 表为空时，程序会用 `GetDefaultPrice()` 的内置价格初始化，与上游行为一致。内置值只是**占位默认值**：不同服务商的采购价差异很大，请在「模型价格」里按自己的实际成本调整，改过的价格行后续升级不会被覆盖。

> 注意：迁移只在 `prices` 表非空时补缺失的价格行，空表时交给内置价格同步，避免出现「只有一行价格」导致其它内置价格不再初始化。

## 从 one-hub 迁移到本版本

```bash
# 1. 备份（务必执行）
mysqldump -h127.0.0.1 -u<user> -p<pass> onehub > onehub_before_fork.sql

# 2. 换镜像，config.yaml 沿用原文件；新配置项全部有默认值，可以不改
docker stop one-hub && docker rename one-hub one-hub-prev
docker run -d --name one-hub --network host   -v /path/to/data:/data ghcr.io/addy163/one-hub:latest
```

首次启动会自动执行迁移，日志里可以确认：

```
database migration started
已补充默认配置项: RelayEndpoints
已补充内置价格行: qwen4-max
已补充模型规格 104 条
database migrated
```

升级后自查清单：

1. 后台「设置 → 运营设置」出现「线路管理（多地址分发）」面板；
2. 「模型价格」能正常打开，内置价格已初始化（可按自己的成本调整）；
3. 「模型信息」有 104 条记录（老库之前已导入过则不会重复插入）；
4. 前台令牌页的地址选择框在配置了多条线路后出现。

回滚：停掉新容器、`docker rename` 回旧容器，再 `mysql onehub < onehub_before_fork.sql` 恢复数据。

## 功能变化


- 全新的 UI 界面
- 新增用户仪表盘
- 新增管理员分析数据统计界面
- 重构了中转`供应商`模块
- 支持使用`Azure Speech`模拟`TTS`功能
- 渠道可配置单独的 http/socks5 代理
- 支持动态返回用户模型列表
- 支持自定义测速模型
- 日志增加请求耗时
- 支持和优化非 OpenAI 模型的函数调用（支持的模型可以在 lobe-chat 直接使用）
- 支持完成倍率自定义
- 支持完整的分页和排序
- 支持`Telegram bot`
- 支持模型按次收费
- 支持模型通配符
- 支持使用配置文件启动程序
- 支持模型价格更新
- 支持自动获取供应商模型
- 支持仅聊天，开启后如果有传入`function call`参数会跳过该渠道
- 支持支付
- 支持配置用户组 RPM
- 支持`Prometheus`监控
- 支持`Uptime Kuma`状态监控(通过环境变量或者配置文件开启)
- 支持用户分组自动升级
- 支持用户月度账单生成(通过环境变量或者配置文件开启)
- 支持多策略(`覆盖更新`,`只更新现有价格`,`只新增价格`)模型价格自动更新(通过环境变量或者配置文件开启）
- 支持`Gemini`、`Claude`格式API请求（详见[文档](https://one-hub-doc.vercel.app/)）
- 支持`Webauthn`登录（前往`个人设置`页面注册Webauthn登陆凭证即可）

## 当前支持的供应商

| 供应商                                                                | Chat                     | Embeddings | Audio  | Images      | 其他                                                             |
| --------------------------------------------------------------------- | ------------------------ | ---------- | ------ | ----------- | ---------------------------------------------------------------- |
| [OpenAI](https://platform.openai.com/docs/api-reference/introduction) | ✅                       | ✅         | ✅     | ✅          | -                                                                |
| [Azure OpenAI](https://oai.azure.com/)                                | ✅                       | ✅         | ✅     | ✅          | -                                                                |
| [Azure Speech](https://portal.azure.com/)                             | -                        | -          | ⚠️ tts | -           | -                                                                |
| [Anthropic](https://www.anthropic.com/)                               | ✅                       | -          | -      | -           | -                                                                |
| [Gemini](https://aistudio.google.com/)                                | ✅                       | -          | -      | -           | -                                                                |
| [百度文心](https://console.bce.baidu.com/qianfan/overview)            | ✅                       | ✅         | -      | -           | -                                                                |
| [通义千问](https://dashscope.console.aliyun.com/overview)             | ✅                       | ✅         | -      | -           | -                                                                |
| [讯飞星火](https://console.xfyun.cn/)                                 | ✅                       | -          | -      | -           | -                                                                |
| [智谱](https://open.bigmodel.cn/overview)                             | ✅                       | ✅         | -      | ⚠️ 图片生成 | -                                                                |
| [腾讯混元](https://cloud.tencent.com/product/hunyuan)                 | ✅                       | -          | -      | -           | -                                                                |
| [百川](https://platform.baichuan-ai.com/console/apikey)               | ✅                       | ✅         | -      | -           | -                                                                |
| [MiniMax](https://www.minimaxi.com/user-center/basic-information)     | ✅                       | ✅         | -      | -           | -                                                                |
| [Deepseek](https://platform.deepseek.com/usage)                       | ✅                       | -          | -      | -           | -                                                                |
| [Moonshot](https://moonshot.ai/)                                      | ✅                       | -          | -      | -           | -                                                                |
| [Mistral](https://mistral.ai/)                                        | ✅                       | ✅         | -      | -           | -                                                                |
| [Groq](https://console.groq.com/keys)                                 | ✅                       | -          | -      | -           | -                                                                |
| [Amazon Bedrock](https://console.aws.amazon.com/bedrock/home)         | ⚠️ 仅支持 Anthropic 模型 | -          | -      | -           | -                                                                |
| [零一万物](https://platform.lingyiwanwu.com/details)                  | ✅                       | -          | -      | -           | -                                                                |
| [Cloudflare AI](https://ai.cloudflare.com/)                           | ✅                       | -          | ⚠️ stt | ⚠️ 图片生成 | -                                                                |
| [Midjourney](https://www.midjourney.com/)                             | -                        | -          | -      | -           | [midjourney-proxy](https://github.com/novicezk/midjourney-proxy) |
| [Cohere](https://cohere.com/)                                         | ✅                       | -          | -      | -           | -                                                                |
| [Stability AI](https://platform.stability.ai/account/credits)         | -                        | -          | -      | ⚠️ 图片生成 | -                                                                |
| [Coze](https://www.coze.com/open/docs/chat?_lang=zh)                  | ✅                       | -          | -      | -           | -                                                                |
| [Ollama](https://github.com/ollama/ollama)                            | ✅                       | ✅         | -      | -           | -                                                                |
| [Suno](https://suno.com/)                                             | -                        | -          | -      | -           | [Suno-API](https://github.com/Suno-API/Suno-API)                 |

## 感谢

- 本程序使用了以下开源项目
  - [one-api](https://github.com/songquanpeng/one-api)为本项目的基础
  - [Berry Free React Admin Template](https://github.com/codedthemes/berry-free-react-admin-template)为本项目的前端界面
  - [minimal-ui-kit](https://github.com/minimal-ui-kit/material-kit-react),使用了其中的部分样式
  - [new api](https://github.com/Calcium-Ion/new-api)，Midjourney/Suno 模块的代码来源于此
  - [go-zero](https://github.com/zeromicro/go-zero) - Token 限流器的实现

感谢以上项目的作者和贡献者

## 交流群

<img src="https://github.com/user-attachments/assets/d1395dac-bc97-481d-af40-ac47e6a00158" width="300">

## 其他

<a href="https://next.ossinsight.io/widgets/official/analyze-repo-stars-history?repo_id=689214770" target="_blank" style="display: block" align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://next.ossinsight.io/widgets/official/analyze-repo-stars-history/thumbnail.png?repo_id=689214770&image_size=auto&color_scheme=dark" width="721" height="auto">
    <img alt="Star History of MartialBE/one-api" src="https://next.ossinsight.io/widgets/official/analyze-repo-stars-history/thumbnail.png?repo_id=689214770&image_size=auto&color_scheme=light" width="721" height="auto">
  </picture>
</a>
