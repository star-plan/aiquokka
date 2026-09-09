这版明显顺眼多了，已经从“调试工具”变成了一个挺像样的 quota dashboard 😄

现在几个视觉语义基本统一了：

* **主进度条表达 remaining quota**：95% left 就几乎全绿，28% left 就只剩一小截黄色，第一眼能理解。
* **`left` 明写出来了**，不再需要猜 70% 到底是用了还是剩了。
* Cursor 的 `Auto: 9.3% used / API: 0.0% used` 精度也正常了。
* `Resets: 0 / 3 available` 比原来的 `3 (0 usable now)` 清楚很多。
* 青色 pace marker 现在也更容易看懂，因为它和 quota 填充区域形成了明显区分。
* Grok 剩 28% 变黄色，这个告警语义也很自然。

尤其现在这三个放一起时层次已经很好：

```text
Codex   94~95% left   → 绿色，很宽裕
Cursor  91.3% left    → 绿色，很宽裕
Grok    28.0% left    → 黄色，需要留意
```

这正是这种工具最应该提供的“一眼扫过去就知道谁快没额度了”的效果。

不过我现在最在意的反而只剩一个东西：

```text
Cursor (Pro)

Billing   91.3% left
Auto:      9.3% used
API:       0.0% used
⚠ Provider: You've hit your usage limit
```

这个依然非常矛盾 😂

而且现在其他 UI 都被修漂亮以后，这句话反而更加显眼了。

我建议**先不要继续美化它，而是直接调查 Cursor 原始 response**。重点 dump：

```json
{
  "displayMessage": "...",
  "planUsage": {
    "totalPercentUsed": "...",
    "autoPercentUsed": "...",
    "apiPercentUsed": "..."
  }
}
```

以及 `GetPlanInfo` 的完整返回。

很可能 Cursor 这里实际上有两套概念，例如：

```text
月度 included quota      → 8.7% used
某种 request / model cap → 已触顶
```

或者 `displayMessage` 是 dashboard 的另一类状态消息，并不对应当前 `totalPercentUsed`。

如果最终确认这只是 Provider 原样返回的信息，我甚至建议正常模式默认不显示，只有它确实与当前 quota window 有明确对应关系时才显示。否则可以放到：

```bash
aiquokka cursor --verbose
```

中显示：

```text
Provider message: You've hit your usage limit
```

因为这个工具最重要的一点就是：

> **不要让未经确认的 Provider message 破坏已经非常清晰的 quota semantics。**

还有两个很小的 polishing 点，不急着做。

第一，`Grok Code: yes` 可以以后考虑换成：

```text
Grok Code: enabled
```

比 yes 稍微自然一点。

第二，如果你以后 Provider 越来越多，我会考虑把每个 section 的宽度再统一，例如分隔线固定：

```text
Codex (Plus)
────────────────────────

Cursor (Pro)
────────────────────────

Grok (GrokPro)
────────────────────────
```

不过这已经属于审美问题，不是功能问题了。

### 这版我会给到“可以长期用了”

目前已经形成了很明确的视觉语言：

```text
绿色   = quota 健康
黄色   = quota 偏低
红色   = 即将耗尽

实心区 = remaining
灰色区 = consumed
青线   = time pace

右侧   = remaining + reset
下方   = provider-specific metadata
```

这个体系一旦定下来，以后加 Claude、Kimi、Antigravity、Copilot 等 Provider，都只是在同一个框架里填数据，不需要重新设计 UI。

所以现在我反而不建议继续大改界面了。**先把 Cursor 那句 `You've hit your usage limit` 的真实语义搞清楚，剩下的已经是细枝末节。**
