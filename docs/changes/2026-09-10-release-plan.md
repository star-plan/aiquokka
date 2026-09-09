# aiquokka v0.1.0 Release Plan

## 目标

为 `star-plan/aiquokka` 准备首次正式版本 `v0.1.0`。

最终应支持：

```text
GitHub Release
├── Windows amd64 / arm64
├── macOS amd64 / arm64
├── Linux amd64 / arm64
└── SHA256SUMS

Windows
└── scoop install aiquokka

macOS / Linux
└── brew install aiquokka

Go
└── go install github.com/star-plan/aiquokka@latest
```

本次任务重点是 **release engineering**，不要继续增加 Provider 或进行无关架构重构。

---

## Phase 1 — Release 前源码整理

### 1.1 修改 Go module identity

项目已经作为 `star-plan/aiquokka` 独立维护，因此：

```go
module github.com/star-plan/aiquokka
```

替换当前：

```go
module github.com/McKean/aiquokka
```

同步修改项目所有内部 import：

```text
github.com/McKean/aiquokka/...
→
github.com/star-plan/aiquokka/...
```

完成后执行：

```bash
go mod tidy
go test ./...
go vet ./...
go build ./...
```

不得使用 `replace` 临时掩盖 module path 问题。

保留原项目 MIT License 及原作者 copyright。当前 LICENSE 是 Christopher Scott 的 MIT License，不要覆盖或删除。

可在 README 简短注明：

```text
Forked from McKean/aiquokka and maintained by star-plan.
```

---

### 1.2 增加版本信息

增加统一 build info，例如：

```text
internal/buildinfo/
└── buildinfo.go
```

建议：

```go
var (
    Version = "dev"
    Commit  = "unknown"
    Date    = "unknown"
)
```

Cobra root 支持：

```bash
aiquokka --version
```

正式 release 应类似输出：

```text
aiquokka 0.1.0
commit abc1234
built 2026-09-10
```

版本信息必须由 release build 通过 `-ldflags` 注入，源码中不要手工维护版本号。

开发构建：

```text
aiquokka dev
```

即可。

---

### 1.3 修正 README

README 必须与当前 fork 实现一致。

重点修改：

#### Provider 列表加入 Cursor

第一屏应至少体现：

```text
Claude · Codex · Cursor · Grok · Kimi · Copilot · ...
```

Usage 加入：

```bash
aiquokka cursor
```

“How it works” 表格加入 Cursor credential discovery 和 Cursor Dashboard API。

#### 修正 CredentialPolicy 描述

删除现在类似：

```text
Auto token refresh — expired OAuth tokens are refreshed and written back.
```

这种已经不准确的描述。

当前真实语义是：

```text
readonly   default
memory     explicit opt-in, only where provider safely supports it
persist    explicit opt-in
```

默认策略必须明确写成：

> aiquokka is a credential consumer by default. Official CLIs own credentials; aiquokka does not refresh or modify them unless explicitly requested.

这才和当前代码一致。

#### 安装说明最终改为

```bash
# Homebrew
brew tap star-plan/tap
brew install aiquokka
```

```powershell
# Scoop
scoop bucket add star-plan https://github.com/star-plan/scoop
scoop install aiquokka
```

以及：

```bash
go install github.com/star-plan/aiquokka@latest
```

---

## Phase 2 — Release Pipeline

### 2.1 Artifact contract

第一版固定以下文件名，不要随意变化，因为 Scoop/Homebrew 都会依赖：

```text
aiquokka-windows-amd64.exe
aiquokka-windows-arm64.exe

aiquokka-macos-amd64
aiquokka-macos-arm64

aiquokka-linux-amd64
aiquokka-linux-arm64

SHA256SUMS
```

所有二进制都必须来自同一个 tag/commit。

---

### 2.2 增加 GoReleaser

建议增加：

```text
.goreleaser.yaml
.github/workflows/release.yml
```

GoReleaser 只负责：

```text
compile
cross-platform artifacts
version ldflags
SHA256SUMS
GitHub Release
```

**不要让 GoReleaser 修改 Scoop / Homebrew 仓库。**

分发仓库继续由现有中央同步机制维护。

Release Workflow：

```text
push tag v*
    ↓
go test ./...
    ↓
GoReleaser
    ↓
GitHub Release
```

Go 版本优先读取：

```yaml
go-version-file: go.mod
```

Release 前至少保证以下目标都能成功构建：

```text
windows/amd64
windows/arm64
darwin/amd64
darwin/arm64
linux/amd64
linux/arm64
```

优先验证：

```text
CGO_ENABLED=0
```

如果某个平台失败，应调查原因，不要为了完成 release 静默删除目标架构。

---

## Phase 3 — CI / Release Gate

正式打 tag 前必须通过：

```bash
go mod tidy
git diff --exit-code

go test ./...
go vet ./...
go build ./...
```

建议额外：

```bash
go test -race ./...
```

并做 CLI smoke test：

```bash
./aiquokka --help
./aiquokka --version
./aiquokka --list
./aiquokka --json
./aiquokka --yaml
```

涉及真实 Provider 的命令允许因为没有 credential 而报告未配置，但不得 panic。

特别确认：

```text
readonly 是默认 credential policy
```

并确保现有 credential safety tests 全部通过。

---

## Phase 4 — Scoop 支持

目标仓库：

```text
star-plan/scoop
```

现有 bucket 已采用 GitHub Release → manifest 的集中同步模式，并且 Workflow 每 4 小时执行一次，同时支持手动触发。

增加：

```text
bucket/aiquokka.json
```

以及：

```text
sync_aiquokka()
```

到：

```text
scripts/sync-manifests.sh
```

支持：

```text
windows-amd64
windows-arm64
```

Manifest 应包含：

```text
version
description
homepage
license = MIT
architecture
hash
bin = aiquokka.exe
checkver
autoupdate
```

SHA256 必须读取 aiquokka Release 的：

```text
SHA256SUMS
```

不要在 app repo 中维护 Scoop manifest。

---

## Phase 5 — Homebrew 支持

目标：

```text
star-plan/homebrew-tap
```

当前 tap 同样由 GitHub Release 自动生成 formula，并有每 4 小时一次的同步 Workflow。

增加：

```text
aiquokka.rb
```

以及：

```text
sync_aiquokka()
```

到：

```text
scripts/sync-formulas.sh
```

支持：

```text
macOS arm64
macOS amd64
Linux arm64
Linux amd64
```

Formula test：

```ruby
test do
  assert_match version.to_s, shell_output("#{bin}/aiquokka --version")
end
```

不要让 aiquokka repo 自己维护 formula。

---

# Phase 6 — Dry Run

在打正式 tag 前执行一次 GoReleaser dry run。

确认生成目录中恰好存在：

```text
6 binaries
1 SHA256SUMS
```

检查：

```bash
file <artifact>
sha256sum -c SHA256SUMS
```

至少本机实际执行当前平台 binary：

```bash
./aiquokka --version
./aiquokka --list
./aiquokka
```

确认版本信息不是：

```text
dev
unknown
```

---

# Phase 7 — 发布 v0.1.0

这是一个 **release gate**。

完成前面的所有修改、测试和 dry run 后：

> 不要立即 tag。先把最终 diff 和测试结果交给用户确认。

获得确认后：

```bash
git status
git log -1 --oneline
git push origin main
```

然后：

```bash
git tag -a v0.1.0 -m "aiquokka v0.1.0"
git push origin v0.1.0
```

等待 Release Workflow。

可以用 GH CLI：

```bash
gh run list \
  -R star-plan/aiquokka \
  --workflow release.yml
```

然后：

```bash
gh run watch <RUN_ID> \
  -R star-plan/aiquokka
```

确认：

```bash
gh release view v0.1.0 \
  -R star-plan/aiquokka
```

当前 `star-plan/aiquokka` 还没有任何 GitHub Release，所以 `v0.1.0` 将是首次正式 Release。

---

# Phase 8 — Release Notes

建议 `v0.1.0` release notes 重点只写用户可感知内容：

```markdown
## aiquokka v0.1.0

First star-plan release of aiquokka.

### Highlights

- Unified quota monitoring for Claude, Codex, Cursor, Grok and more
- New Cursor provider with Agent/Desktop credential discovery
- Remaining-quota bars with pace indicators
- Provider registry architecture
- Safe credential handling with read-only defaults
- JSON/YAML and live watch modes
- Windows, macOS and Linux binaries

### Installation

Homebrew:

    brew tap star-plan/tap
    brew install aiquokka

Scoop:

    scoop bucket add star-plan https://github.com/star-plan/scoop
    scoop install aiquokka
```

同时保留 upstream attribution。

---

# Phase 9 — 同步 Scoop / Homebrew

GitHub Release 发布成功以后，先不要等 4 小时 cron。

直接使用本机已经认证的 GH CLI 手动启动两个 Workflow：

```bash
gh workflow run sync.yml \
  -R star-plan/scoop
```

```bash
gh workflow run sync.yml \
  -R star-plan/homebrew-tap
```

查看：

```bash
gh run list -R star-plan/scoop --workflow sync.yml
gh run list -R star-plan/homebrew-tap --workflow sync.yml
```

分别：

```bash
gh run watch <RUN_ID> -R star-plan/scoop
gh run watch <RUN_ID> -R star-plan/homebrew-tap
```

这样第一版不需要为了 cross-repository dispatch 再配置 PAT/GitHub App。

现有定时任务继续作为兜底即可。

---

# Phase 10 — 安装验收

### Windows / Scoop

在 Windows 实机：

```powershell
scoop update
scoop bucket add star-plan https://github.com/star-plan/scoop
scoop install aiquokka
```

检查：

```powershell
where.exe aiquokka
aiquokka --version
aiquokka --list
aiquokka
```

再测试：

```powershell
scoop update aiquokka
```

---

### macOS / Homebrew

```bash
brew update
brew tap star-plan/tap
brew install aiquokka
```

检查：

```bash
which aiquokka
aiquokka --version
aiquokka --list
```

---

### Linux / Homebrew

至少验证 Linux amd64：

```bash
brew install star-plan/tap/aiquokka
aiquokka --version
```

---

### Go install

最后验证：

```bash
go install github.com/star-plan/aiquokka@v0.1.0
```

以及：

```bash
go install github.com/star-plan/aiquokka@latest
```

二者都必须成功。

---

# Definition of Done

Agent 只有在以下全部满足后才能宣布 release 完成：

```text
[ ] module path 已迁移到 github.com/star-plan/aiquokka
[ ] 所有 imports 已迁移
[ ] README 与 Cursor / CredentialPolicy 实现一致
[ ] aiquokka --version 可用
[ ] go test ./... 通过
[ ] go vet ./... 通过
[ ] 六个平台/架构 build 成功
[ ] SHA256SUMS 完整
[ ] v0.1.0 GitHub Release 发布成功
[ ] Scoop manifest 已同步
[ ] Homebrew formula 已同步
[ ] Scoop 实机安装成功
[ ] Homebrew 至少一个平台实机安装成功
[ ] go install github.com/star-plan/aiquokka@latest 成功
```

## 操作约束

Agent 可以自由使用：

```bash
git
gh
go
goreleaser
```

但：

> **在创建 `v0.1.0` tag 之前必须停止并向用户汇报：修改内容、测试结果、dry-run artifact 列表。未经确认不得 push release tag。**

这条我建议保留。前面的代码修改即使有问题还能继续 amend；**tag + GitHub Release + Scoop/Homebrew 同步之后就进入公共发布状态了**，这里设一个人工 gate 很值。
