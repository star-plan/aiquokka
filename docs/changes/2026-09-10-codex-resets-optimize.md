刚查了 Codex 现在的接口实现，**这不是“我们猜测可以做”，而是后端确实提供了每一张 reset credit 的明细和过期时间**。

你现在 `aiquokka` 只请求：

```text
GET /backend-api/wham/usage
```

并且目前只解析：

```go
rate_limit_reset_credits: {
    available_count,
    applicable_available_count,
}
```

所以现阶段只能显示截图里的：

```text
Resets: 0 / 3 available
```

你的代码现在确实就是这样实现的。

但 Codex 还有一个专门的接口：

```text
GET https://chatgpt.com/backend-api/wham/rate-limit-reset-credits
```

OpenAI 自己的 Codex 代码已经在调用这个 endpoint，而且 protocol 里明确有每张 credit 的数据结构。([GitHub][1])

返回的每张 reset 大致有：

```json
{
  "id": "credit-...",
  "reset_type": "codex_rate_limits",
  "status": "available",
  "granted_at": "...",
  "expires_at": "...",
  "title": "Full reset (Weekly + 5 hr)",
  "description": "..."
}
```

其中最有价值的就是：

```text
expires_at
```

而且 OpenAI 自己的测试 fixture 里甚至直接测试了多张 reset credit、不同 `granted_at` 和 `expires_at` 的情况。([GitHub][2])

所以我觉得**非常值得加**。

---

不过我不会把 UI 做得太啰嗦。

你这个工具最大的优点现在就是：

> 打开一眼扫完所有订阅。

如果每张 reset 都打印五六个字段，Codex 一下占半屏，就破坏这个优势了。

我会做成：

```text
Codex (plus)
────────────────────────────────

  5h       [█████░░░░░░░░]  23.0% left   resets in 2h32m
  Weekly   [████████████░░]  67.0% left   resets in 4d17h

  Resets:  3 banked · 0 usable now
    #1     expires in 6d3h   (Sep 16 22:00)
    #2     expires in 13d3h  (Sep 23 22:00)
    #3     expires in 20d3h  (Sep 30 22:00)
```

我现在甚至会把你当前的：

```text
Resets: 0 / 3 available
```

再改一次。

因为看过 API 字段以后，这句话其实**语义不够准确**：

```text
available_count              = 3
applicable_available_count   = 0
```

这里不是：

> 3 个里面只有 0 个 available

而更接近：

> **有 3 个已经 banked 的 reset credits，但目前没有一个适用于当前状态。**

所以：

```text
3 banked · 0 usable now
```

其实比：

```text
0 / 3 available
```

更忠于后端语义。

之前我们只从 UI 可读性看，觉得 `0 / 3 available` 更简洁；现在拿到这个接口语义以后，我会推翻那一小点建议 😂。

---

### 还能做得更聪明一点

平时如果只有一张：

```text
Resets: 1 banked · expires in 5d
```

不用展开。

两三张的时候：

```text
Resets: 3 banked · 0 usable now
  expires: Sep 16 · Sep 23 · Sep 30
```

其实已经足够。

只有：

```bash
aiquokka codex --verbose
```

或者 `--json` 时，才完整暴露：

```json
"reset_credits": [
  {
    "id": "...",
    "status": "available",
    "reset_type": "codex_rate_limits",
    "granted_at": "...",
    "expires_at": "...",
    "title": "..."
  }
]
```

这样我觉得最符合 AIQuokka 现在的产品定位：

> **默认 glanceable，结构化输出完整。**

---

还有一个实现细节值得注意。

`/wham/usage` 的：

```text
rate_limit_reset_credits.available_count
```

只是 **summary**。

真正每张 credit 的 `expires_at` 必须通过 dedicated endpoint 拿。其他已经实现这个功能的工具也明确指出这一点。([GitHub][3])

所以 Codex Provider 可以变成：

```text
Fetch()
  │
  ├─ GET /wham/usage
  │      ├─ quota windows
  │      ├─ available_count
  │      └─ applicable_available_count
  │
  └─ if available_count > 0
         ↓
     GET /wham/rate-limit-reset-credits
         ↓
     []ResetCredit
```

我会**只有 `available_count > 0` 才发第二个请求**，避免每次 `-w` 刷新都白白多打一遍 API。

---

另外 OpenAI 的 protocol 里还有一个很重要的防御性说明：

> reset credit detail list **可能被后端截断**，所以 `len(credits)` 可以小于 `available_count`。([GitHub][1])

所以千万不要写：

```go
total := len(credits)
```

应该始终：

```go
total := summary.AvailableCount
details := len(credits)
```

例如后端说：

```text
available_count = 10
```

但只返回 5 个详情，就应该显示：

```text
Resets: 10 banked
  showing 5 details
```

而不是误报成 5。

---

我觉得这个功能特别值得做，因为 **reset credits 本身也是一种“会过期的 AI 订阅资源”**，完全符合 AIQuokka 的核心使命。

现在你已经能一眼看到：

```text
Codex quota
Cursor quota
Grok quota
```

再加：

```text
Codex reset credits × 3
  └─ 每个什么时候过期
```

以后就不会出现那种：

> “卧槽，原来我还有三个 reset，结果放着放着过期了”

的情况 😂。


[1]: https://github.com/openai/codex/blob/main/codex-rs/app-server-protocol/src/protocol/v2/account.rs "https://github.com/openai/codex/blob/main/codex-rs/app-server-protocol/src/protocol/v2/account.rs"
[2]: https://github.com/openai/codex/blob/main/codex-rs/app-server/tests/suite/v2/rate_limits.rs "https://github.com/openai/codex/blob/main/codex-rs/app-server/tests/suite/v2/rate_limits.rs"
[3]: https://github.com/zlliang/pi-spark/blob/main/docs/openai-codex-banked-rate-limit-resets.md "https://github.com/zlliang/pi-spark/blob/main/docs/openai-codex-banked-rate-limit-resets.md"
