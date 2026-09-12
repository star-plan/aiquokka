## aiquokka credential 自动恢复优化

仓库：`star-plan/aiquokka`

### 背景

目前 Codex / Grok 的 credential 很容易出现 access token 过期。当前表现是直接显示：

```text
credentials expired or unusable — re-login with the official CLI,
or pass --credential-policy persist
```

现有代码其实已经实现了大部分 refresh 基础设施：

* `internal/credential/policy.go`

  * `ReadOnly`
  * `RefreshInMemory`
  * `RefreshAndPersist`
  * 当前默认 `DefaultPolicy = ReadOnly`
* `internal/credential/refresh.go`

  * `Apply()`
  * `TryOfficialRefresh()`
* `internal/credential/capability.go`

  * provider 声明 credential refresh 能力
* Codex：

  * `internal/providers/codex/refresh.go`
  * 已实现 OAuth refresh + persist
  * `RotatesRefreshToken=true`
  * 当前主要在 API 401 后触发 refresh
* Grok：

  * `internal/providers/grok/refresh.go`
  * 已实现 refresh + persist
  * 可以根据 `expires_at` 主动判断过期
  * 同样会 rotate refresh token
* CLI：

  * `cmd/root.go` 中存在全局 `--credential-policy`
  * 默认 readonly
  * `cmd/run.go` 目前 provider Fetch 出错后基本只是直接向上返回/显示 error

### 修改目标

不要简单实现成：

```text
token expired
→ 直接启动 codex / grok 等 CLI 去刷新 credential
```

希望改成两级 credential recovery：

```text
credential 可用
    ↓
正常查询

access token 过期 / 401
    ↓
尝试静默 refresh
    ↓
refresh 成功
    ↓
重新读取 credential
    ↓
原请求 retry 一次

如果 refresh token 已失效 / missing / invalid_grant
    ↓
标记为 ReauthRequired
    ↓
才考虑启动官方 CLI，让 CLI 自动刷新 credential，或者让用户手动启动官方 CLI
```

核心原则：

> **Refresh 和 Re-authentication 必须区分。**
>
> 普通 access token 过期不应该要求用户重新登录。

### 建议增加 `auto` policy

建议把 credential policy 改成：

```text
auto        # 新默认
readonly
memory
persist
```

`auto` 的含义：

* provider credential 正常：不做任何修改
* token 需要 refresh：使用该 provider 声明的安全恢复方式
* refresh 成功：persist 后继续请求
* refresh 无法继续：返回明确的 `ReauthRequired`
* 不应该因为一个 provider 出错而影响其他 provider

保留：

```bash
aiquokka --credential-policy readonly
```

给明确不允许 aiquokka 修改 credential 的用户。

`persist` 继续作为显式强制策略。

### Provider 行为

Codex 和 Grok 当前都：

```go
RefreshInMemory:     false
RefreshAndPersist:   true
RotatesRefreshToken: true
```

因此 `auto` 对这两个 provider 本质应该选择：

```text
refresh + persist
```

而不是 memory refresh。

refresh 成功后必须 retry 原 API **最多一次**，避免无限刷新循环。

对于这些错误：

```text
missing refresh token
invalid_grant
refresh token expired
refresh token revoked
refresh token reused
```

不要继续 retry，应转换为统一的：

```go
ReauthRequiredError
```

最好包含：

```go
Provider
Command
Cause
```

例如：

```text
Codex authentication expired — run `codex login`
Grok authentication expired — run `grok login`
```

### Official CLI re-auth 行为

先不要把「自动启动登录 CLI」和 token refresh 混在 credential.Apply 里面。

建议将其作为独立 recovery 阶段。

行为区分：

```text
aiquokka codex
```

如果：

* stdin/stdout 是 TTY
* 非 JSON/YAML
* 非 watch aggregate
* Codex 返回 ReauthRequired

可以执行：

```bash
codex login
```

登录成功以后：

```text
reload credential
→ Fetch retry 一次
```

Grok 同理：

```bash
grok login
```

但以下情况 **禁止自动启动交互式 CLI**：

```text
aiquokka                  # aggregate
aiquokka --json
aiquokka --yaml
stdout pipe / redirect
CI / non-TTY
watch 自动刷新阶段
```

这些情况下只显示：

```text
Codex
─────
  Sign-in required — run `codex login`
```

不能突然打开浏览器或劫持 terminal。

### Grok 特别注意

Grok refresh token 会 rotate。

目前 `grok/refresh.go` 虽然最终用了 atomic write，但这只能避免文件写坏，**无法解决多个进程同时 refresh 的竞态**。

官方 Grok 目前使用：

```text
~/.grok/auth.json.lock
```

并在 refresh / persist 时做跨进程锁。

因此在把 Grok 自动 refresh 作为默认行为之前，请参考官方 Grok 实现，补上与官方兼容的 `auth.json.lock` 机制。

至少做到：

```text
acquire lock
→ reload auth.json
→ 确认当前 account / refresh token 没被其他进程更新
→ refresh
→ 再次安全 merge
→ atomic persist
→ release lock
```

不要让旧 refresh result 覆盖更新后的 rotating refresh token，否则后面很容易出现 `invalid_grant`。

### 代码结构建议

不要大改现有架构，尽量复用：

```text
internal/credential/
    policy.go
    capability.go
    refresh.go
    errors.go
```

但建议明确区分两个概念：

```text
SilentRefresh
InteractiveReauth
```

不要继续把两者都隐含在 `OfficialCLIRefresh` 里。

可以考虑增加类似：

```go
type RecoveryFuncs struct {
    Refresh func(context.Context) error
    Reauth  func(context.Context) error
}
```

或者在现有 `RefreshFuncs` 基础上最小改动实现。

CLI 层的 interactive reauth 应该放在 `cmd` 层处理，而不是 provider 内部直接 `exec.Command("codex", "login")`。

Provider 负责告诉上层：

```text
I need re-authentication.
```

CLI 层决定：

```text
当前环境是否允许交互登录。
```

### UX 目标

正常情况下：

```bash
$ aiquokka
```

用户不应该看到 token expire。

流程应该静默：

```text
expired
→ refresh
→ persist
→ retry
→ 显示额度
```

只有 refresh token 真正失效才看到：

```text
Codex
─────
  Sign-in required — run `codex login`
```

单独执行：

```bash
aiquokka codex
```

在交互终端里则可以进一步自动进入官方登录流程，成功后继续显示额度。

### 测试要求

至少补充以下测试：

```text
1. auto + valid token → 不触发 refresh
2. auto + expired token → refresh + persist + retry 成功
3. refresh 后第二次仍然 401 → 不无限 retry
4. invalid_grant → ReauthRequiredError
5. readonly → 保持现在不修改 credential 的行为
6. aggregate 模式出现 ReauthRequired → 其他 provider 正常显示
7. JSON/YAML / non-TTY → 永远不启动 interactive login
8. single provider + interactive TTY → reauth 成功后重新 Fetch
9. Grok rotating refresh token 不会被并发旧写覆盖
```

### 本次修改优先级

优先完成：

```text
1. auto credential recovery
2. RefreshRequired / ReauthRequired 状态区分
3. Codex / Grok silent refresh + retry once
4. Grok refresh locking
5. CLI 层 interactive reauth fallback
6. 更新 README / --help
```

尽量保持当前 provider API 和渲染逻辑不变，不做无关重构。
