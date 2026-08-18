<p align="right">
  <strong>简体中文</strong> | <a href="./README.en.md">English</a>
</p>

# CodexSwitch

<p align="center">
  <strong>Codex 官方账号与 OpenAI API 配置切换桌面工具</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Desktop-Wails%20App-2f855a?style=for-the-badge" alt="Desktop App" />
  <img src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.26+" />
  <img src="https://img.shields.io/badge/Node.js-22.x-339933?style=for-the-badge&logo=node.js&logoColor=white" alt="Node.js 22.x" />
  <img src="https://img.shields.io/badge/Vue-3.5-4FC08D?style=for-the-badge&logo=vue.js&logoColor=white" alt="Vue 3.5" />
  <img src="https://img.shields.io/badge/TypeScript-5.8-3178C6?style=for-the-badge&logo=typescript&logoColor=white" alt="TypeScript 5.8" />
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Platforms-Windows%20%7C%20macOS%20%7C%20Linux-1f2937?style=flat-square" alt="Platforms" />
  <img src="https://img.shields.io/badge/Release-GitHub%20Actions-7c3aed?style=flat-square" alt="GitHub Actions" />
  <img src="https://img.shields.io/badge/Version-1.0.3-2563eb?style=flat-square" alt="Version" />
  <img src="https://img.shields.io/badge/Language-中文%20%7C%20English-ef4444?style=flat-square" alt="Language" />
</p>

<p align="center">
  <a href="https://github.com/fangxielian0116/CodexSwitch/releases"><img src="https://img.shields.io/github/v/release/fangxielian0116/CodexSwitch?style=flat-square" alt="Latest Release" /></a>
  <a href="https://github.com/fangxielian0116/CodexSwitch/stargazers"><img src="https://img.shields.io/github/stars/fangxielian0116/CodexSwitch?style=flat-square" alt="GitHub Stars" /></a>
  <a href="https://github.com/fangxielian0116/CodexSwitch/issues"><img src="https://img.shields.io/github/issues/fangxielian0116/CodexSwitch?style=flat-square" alt="GitHub Issues" /></a>
</p>

<p align="center">
  <a href="https://github.com/fangxielian0116/CodexSwitch/releases">下载发布版</a>
</p>

CodexSwitch 是一个跨平台桌面应用，用来统一管理多个 Codex 官方账号和自定义 OpenAI API 配置。它会连接当前正在使用的 Codex 配置目录与本地托管配置库，让你在不同账号、不同 API Key、不同模型参数之间快速切换，减少手动编辑 `auth.json` 和 `config.toml` 的成本。

> 本仓库基于原项目 [ke4nec/CodexSwitch](https://github.com/ke4nec/CodexSwitch) 继续整理和扩展。感谢原作者提供项目基础、产品方向和开源实现。

## 界面预览

<p align="center">
  <img src="./docs/preview.png" alt="CodexSwitch 界面预览" width="100%" />
</p>

## 本仓库更新重点

当前版本围绕“更稳定地切换配置”和“更清楚地看到账号状态”做了重点补强：

- **官方账号与 API 配置统一托管**
  支持自动识别当前 Codex 目录、导入官方账号文件、新建/编辑 API 配置，并把不同来源的配置统一放进本地托管列表。

- **兼容标准官方账号文件和 CLI 账号文件**
  导入官方账号时会识别标准 `auth.json`，也会兼容包含 `access_token`、`refresh_token`、`account_id` 等字段的 CLI 格式账号文件，并规范化为 Codex 可使用的结构。

- **API 配置生成能力增强**
  新建 API 配置时会生成完整的 `auth.json` 与 `config.toml`，支持填写 `Base URL`、模型、推理强度、上下文窗口，并自动计算 `model_auto_compact_token_limit`。

- **切换前保护当前配置**
  切换到目标配置前，会先扫描并保存当前 Codex 目录中的有效配置，降低误覆盖当前账号或 API 配置的风险。

- **更稳的 `config.toml` 回写**
  官方账号共享一份官方配置模板；切换 API 配置或从 API 切回官方账号时，会把托管配置与目标目录现有配置合并，尽量保留已有设置。

- **官方账号额度刷新**
  支持刷新官方账号额度，解析 5 小时窗口和每周窗口，记录套餐类型、重置时间和刷新状态。刷新前会尝试更新官方 access token。

- **延迟与可用性测试**
  支持对官方账号和 API 配置执行单个或全部测试。API 配置会测试基础连通性，并按 `wire_api` 调用 `/responses` 或 `/chat/completions` 进行可用性验证。

- **系统代理支持**
  Windows 启动时会自动读取当前用户的静态系统代理和绕过列表，并统一用于额度刷新、官方 Token 刷新和可用性测试；未配置系统代理时仍支持标准环境变量代理。

- **API 连通性历史**
  API 配置的延迟和可用性结果会写入本地 SQLite，最多保留 48 条历史记录。界面会用历史点位展示最近连接结果，便于观察服务是否稳定。

- **列表排序和状态展示**
  配置列表支持按 5 小时额度、每周额度、延迟和最后同步时间排序，并展示当前激活、禁用、异常、就绪等状态。

- **系统托盘与快速切换**
  支持系统托盘菜单，托盘中会显示当前配置，并提供打开主窗口、快速切换和退出入口。Windows 下点击关闭按钮可按设置隐藏到托盘。

- **切换后自动重启 Codex**
  可在设置中开启“切换后自动重启 Codex”。当前实现支持 Windows 和 macOS，便于新配置在 Codex 下一次启动时生效。

- **Windows 网络修复**
  Windows 下提供网络修复入口，会以管理员权限调整物理网卡与 VPN/虚拟网卡的 IPv4 metric，清理 DNS 缓存，并检测 `api.openai.com` 与 `chatgpt.com` 的可达性。

- **本地维护能力**
  支持禁用/启用配置、删除未激活配置、设置已归档对话保留天数，并在启动或刷新时清理过期归档内容。

- **中英文界面**
  应用内置中文和英文文案，并会记住用户选择的语言。

## 适合场景

- 经常在多个 Codex 官方账号之间切换
- 同时维护官方账号和多个 OpenAI API Key
- 需要为不同模型、Base URL、推理强度准备独立配置
- 想快速查看官方账号额度、API 可用性和延迟
- 不想反复手动修改 `~/.codex/auth.json` 和 `~/.codex/config.toml`

## 使用流程

1. 启动 CodexSwitch。
2. 在设置中确认目标 Codex 配置目录，默认通常是 `~/.codex` 或 `%USERPROFILE%\.codex`。
3. 让应用自动识别当前配置，或点击“导入账号文件”导入官方账号。
4. 按需新增 API 配置，填写 Base URL、模型、推理强度、上下文窗口和 API Key。
5. 在托管配置列表中执行切换、测试、刷新额度、禁用、删除等操作。
6. 如有需要，在设置中开启“切换后自动重启 Codex”或“关闭按钮隐藏到托盘”。

## 常见 Codex 配置目录

- Windows: `%USERPROFILE%\.codex`
- macOS: `~/.codex`
- Linux: `~/.codex`

目标目录可以在应用设置中手动修改。保存设置后，应用会立即重新扫描并识别当前激活配置。

## 数据与安全说明

- 托管配置保存在本机应用配置目录中，不会上传到云端。
- API Key、官方账号 token 等敏感信息会保存在本地托管配置中，请不要把应用配置目录上传到公开仓库。
- 删除托管配置只会删除 CodexSwitch 管理的副本，不会主动清空目标 Codex 目录。
- 当前激活配置不能直接删除，需要先切换到其他配置。

## 下载与发布

发布工作流位于 [`.github/workflows/release-cross-platform.yml`](.github/workflows/release-cross-platform.yml)，会构建以下平台产物：

- Linux `amd64`
- Windows `amd64`
- macOS `amd64`
- macOS `arm64`

版本号来自 [`wails.json`](wails.json) 的 `info.productVersion`。推荐通过版本 tag 发布：

```bash
git tag v1.0.3
git push origin v1.0.3
```

当前工作流也支持手动触发。按分支触发时，工作流配置监听 `master` 分支上的 [`wails.json`](wails.json) 变更；如果你的默认分支是 `main`，请使用 tag 或手动触发发布，或者同步调整 workflow 分支配置。

## 从源码构建

### 环境要求

- Go `1.26+`
- Node.js `22.x`
- Wails CLI `v2.11.0+`

安装 Wails CLI：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 初始化依赖

```cmd
bootstrap.bat
```

### 调试开发

```cmd
dev.bat
```

### 本地构建

```cmd
build.bat
```

### 本地发布构建

```cmd
release.bat
```

### 清理构建产物

```cmd
clean.bat
```

彻底清理，包括 `node_modules`：

```cmd
clean.bat -All
```

## 手工命令

如果你更习惯直接执行命令：

```bash
cd frontend
npm install
npm run build
cd ..
go test ./...
go build ./...
wails build
```

## 技术栈

- **后端**: Go
- **桌面壳**: Wails v2
- **前端**: Vue 3 + TypeScript + Vuetify + Pinia
- **本地存储**: JSON 文件 + SQLite
- **构建发布**: GitHub Actions

## 项目结构

- [`app.go`](app.go): Wails 绑定方法，连接前端操作和后端服务
- [`internal/codexswitch`](internal/codexswitch): 配置扫描、导入、切换、额度刷新、延迟测试、设置存储等核心逻辑
- [`frontend`](frontend): Vue 桌面界面、状态管理和双语文案
- [`docs/preview.png`](docs/preview.png): README 预览图
- [`.github/workflows/release-cross-platform.yml`](.github/workflows/release-cross-platform.yml): 跨平台发布工作流

## 当前限制

- 网络修复目前仅支持 Windows。
- 自动重启 Codex 目前支持 Windows 和 macOS；Linux 会返回不支持提示。
- 官方账号额度刷新和延迟测试依赖 ChatGPT/Codex 相关接口可访问，接口变动可能导致结果不可用。
- API 可用性测试会发起一次轻量请求，测试提示词为 `hi`。

## 致谢

感谢 [ke4nec/CodexSwitch](https://github.com/ke4nec/CodexSwitch) 原作者提供项目基础和开源实现。本仓库在此基础上继续完善账号导入、API 配置、额度刷新、延迟测试、托盘体验、网络修复和自动化发布等能力。

同时感谢 Wails、Go、Vue、Vuetify、Pinia、SQLite 等开源项目。
