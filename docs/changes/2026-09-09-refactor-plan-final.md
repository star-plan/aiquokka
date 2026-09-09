# aiquokka Refactor Plan

## 目标

在保留现有 **quota model、watch UI、并发查询框架** 的前提下，完成以下改造：

1. Provider 统一注册，避免在 `cmd/root.go` 和独立 Cobra command 中重复硬编码。
2. 明确 credential ownership：**官方 CLI / Provider 客户端负责身份与凭据，aiquokka 默认只读消费凭据。**
3. 引入统一的 Credential Policy / Capability 模型，避免错误刷新或覆盖官方 credential。
4. 新增 Cursor Provider，同时支持 Cursor Agent 与 Desktop credential discovery，并严格使用 Provider 返回的 quota 数据。

---

## 1. Provider Registry

新增统一 Provider 接口与 Registry。Provider 只需实现一次并显式注册一次，CLI、聚合查询、子命令和 `--list` 均从 Registry 自动生成。

```go
type Provider interface {
    ID() string
    Name() string
    Description() string
    Fetch(ctx context.Context) (*usage.Report, error)
}
```

建议目录：

```text
internal/provider/
├── provider.go
└── registry.go

internal/providers/
├── catalog.go
├── claude/
├── codex/
├── cursor/
├── grok/
└── ...
```

`catalog.go` 是唯一 Provider 清单：

```go
func All() []provider.Provider {
    return []provider.Provider{
        claude.New(),
        codex.New(),
        cursor.New(),
        grok.New(),
    }
}
```

CLI 根据 Registry 自动创建：

- `aiquokka`
- `aiquokka <provider>`
- `aiquokka --list`
- `aiquokka --json`
- `aiquokka --watch`

不要使用 `init()` 自注册、运行时目录扫描或 Go plugin；保持显式注册、单一事实来源。

---

## 2. Credential Ownership

原则：

> **Provider 负责 quota；官方 CLI / 客户端负责 identity。aiquokka 默认只是 credential consumer。**

aiquokka 不应默认刷新或修改官方 credential。若官方 CLI 能自行处理刷新，应优先调用官方 CLI，再重新读取 credential。

### Policy

```go
type Policy uint8

const (
    ReadOnly Policy = iota
    RefreshInMemory
    RefreshAndPersist
)
```

语义：

- `ReadOnly`：aiquokka 不直接 refresh 或写 credential；允许调用官方 CLI，由官方 CLI 自行处理 credential。
- `RefreshInMemory`：aiquokka 可自行 refresh，但不得写回 credential store。
- `RefreshAndPersist`：aiquokka 可自行 refresh，并原子写回 credential store。

默认策略：`ReadOnly`。

### Capability

Policy 由用户选择；Provider 只声明自己安全支持哪些行为。

```go
type Capabilities struct {
    OfficialCLIRefresh  bool
    RefreshInMemory     bool
    RefreshAndPersist   bool
    RotatesRefreshToken bool
}
```

规则：

- 不确定 refresh token 是否 rotation 时，默认 `RefreshInMemory = false`。
- 若 refresh 会 rotate token，则禁止只在内存刷新后丢弃新 refresh token。
- 持久化必须使用安全、原子的写入方式。
- Policy 超出 Provider capability 时应直接报错，不得静默降级。

推荐 flow：

```text
Fetch quota
   │
credential valid?
   ├─ yes → query
   └─ no
       ↓
TryOfficialRefresh()   # capability 允许时
       ↓
reload credential
       ↓
credential valid?
   ├─ yes → query
   └─ no  → apply user Policy
```

建议新增：

```text
internal/credential/
├── policy.go
├── capability.go
├── errors.go
└── refresh.go
```

---

## 3. Cursor Provider

第一版目标：支持 Cursor Agent + Cursor Desktop，并复用 Cursor dashboard API。

### Credential discovery

优先级：

1. Cursor Agent auth file
2. OS credential store
3. Cursor Desktop `state.vscdb`

已知来源：

| 平台 | Cursor Agent | Fallback |
| --- | --- | --- |
| Linux | `$XDG_CONFIG_HOME/cursor/auth.json`，默认 `~/.config/cursor/auth.json` | `secret-tool` |
| Windows | `%APPDATA%\Cursor\auth.json` | Credential Manager |
| macOS | Login Keychain | `~/.cursor/auth.json` |

Desktop fallback：从 `state.vscdb` 读取 `cursorAuth/accessToken` 与 `cursorAuth/refreshToken`。

建议结构：

```text
internal/providers/cursor/
├── cursor.go
├── api.go
├── credentials.go
├── auth_file.go
├── desktop_db.go
├── keychain_darwin.go
├── keychain_windows.go
├── keychain_linux.go
└── cursor_test.go
```

Credential 必须成对保存来源，禁止跨来源拼接 access token / refresh token：

```go
type Credential struct {
    AccessToken  string
    RefreshToken string
    Source       CredentialSource
}
```

Resolver 建议返回 `[]Credential`，按优先级逐个尝试。调试模式应显示实际使用的 source；正常输出不显示。

### Cursor quota semantics

调用：

```text
POST https://api2.cursor.sh/aiserver.v1.DashboardService/GetCurrentPeriodUsage
POST https://api2.cursor.sh/aiserver.v1.DashboardService/GetPlanInfo
```

请求头：

```http
Authorization: Bearer <token>
Content-Type: application/json
Connect-Protocol-Version: 1
```

必须遵守：

> **quota percentage 优先使用 Provider 明确返回的值；缺失时返回 unknown，不自行猜测。**

Cursor 的权威百分比为：

```text
planUsage.totalPercentUsed
```

不要用以下字段重建 quota percentage：

```text
includedSpend / limit
```

若 `totalPercentUsed` 缺失，则 `UsedPercent == nil`。

建议 Report 映射：

```go
Report{
    Provider: "Cursor",
    Plan:     plan,
    Windows: []usage.Window{
        {
            Label:       "Billing",
            UsedPercent: totalPercentUsed,
            ResetsAt:    billingCycleEnd,
            Duration:    billingCycleEnd.Sub(billingCycleStart),
        },
    },
}
```

`autoPercentUsed`、`apiPercentUsed`、On-Demand spending 等放入 `Extra`，不要与 included quota 合并为一个百分比。

---

## 4. 必须覆盖的测试

至少添加以下测试：

1. `CursorUsesProviderReportedPercent`
   - `totalPercentUsed = 2.9`
   - 即使 `includedSpend / limit = 50%`
   - 结果仍必须为 `2.9`。

2. `CursorDoesNotGuessMissingPercentage`
   - 缺少 `totalPercentUsed`
   - `UsedPercent == nil`。

3. Credential source precedence
   - Agent + Desktop 同时存在 → Agent 优先。
   - Agent 缺失 → Desktop fallback。

4. ReadOnly safety
   - expired credential + `ReadOnly`
   - credential store 必须保持 byte-for-byte unchanged。

5. Rotating refresh-token safety
   - Provider 声明 `RotatesRefreshToken = true`
   - 用户选择 `RefreshInMemory`
   - 必须拒绝并返回明确错误。

---

## 5. 实施顺序

1. 引入 `internal/provider` Registry，移除重复 Provider/Cobra 注册逻辑。
2. 引入 `internal/credential`：Policy、Capabilities、统一错误类型。
3. 先迁移 Grok，用 rotating refresh token 验证 Credential Policy。
4. 迁移 Codex / Claude，默认改为 `ReadOnly`。
5. Kiro 保持官方 CLI ownership 模式。
6. 新增 Cursor Provider：
   - Agent auth discovery
   - OS credential store
   - Desktop DB fallback
   - `GetCurrentPeriodUsage`
   - `GetPlanInfo`
   - authoritative `totalPercentUsed`
7. 最后增加全局 CLI 参数：

```bash
aiquokka --credential-policy readonly   # default
aiquokka --credential-policy memory
aiquokka --credential-policy persist
```

第一阶段不实现 per-provider policy override；有实际需求后再扩展，例如：

```bash
aiquokka --credential-policy grok=readonly
```

---

## 6. 非目标

本轮不要：

- 重写现有 quota model、watch UI 或并发框架。
- 做动态 plugin / runtime package discovery。
- 为 Cursor 反向推算 Provider 未明确返回的 quota percentage。
- 默认修改官方 CLI credential。
- 过早引入 per-provider 配置系统。

---

## Reference

- aiquokka: https://github.com/McKean/aiquokka
- ai-quota: https://github.com/hunterzhang86/ai-quota (已经clone到 temp/ai-quota)
- usagebat: https://github.com/yutat23/usagebat (已经clone到 temp/usagebat)
- cursor-pulse API notes: https://github.com/cnwinds/cursor-pulse/blob/master/docs/cursor-usage-api.md

