# LumorPuls

单节点 AI 生态**变化监控**：定时用 `agent-browser` 抓页面 → 与上次快照比对 → LLM 生成 Signal → MySQL 存储 → HTTP 输出。

## 前置

1. **MySQL**：创建库 → `go run .` 一次（建表）→ 导入任务：
   ```bash
   mysql -u root -p lumor_puls < scripts/seed_tasks.sql
   ```
 
2. **agent-browser**（PowerShell 不要用 `&&`）：
   ```powershell
   npm install -g agent-browser
   ```

   **推荐：用本机已安装的 Chrome，跳过 install**，在 `config.json` 配置：
   ```json
   "executablePath": "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"
   ```
   验证：
   ```powershell
   $env:AGENT_BROWSER_EXECUTABLE_PATH = "C:\Program Files\Google\Chrome\Application\chrome.exe"
   agent-browser open https://example.com
   agent-browser close
   ```
 
3. **LLM**：在 `config.json` 填 `llm.apiKey`，或环境变量 `LUMOR_LLM_API_KEY`
4. **`browser.waitNetworkIdle`**：建议保持 `false`。OpenAI 等 SPA 用 `networkidle` 可能一直等不到结束。

## 配置



| 项 | 作用 |
|----|------|
| `mysqlDsn` | 数据库 |
| `scheduler` | 是否常驻调度、`tickSec` |
| `browser` | agent-browser 路径、本机 Chrome（`executablePath`） |
| `llm` | DeepSeek/OpenAI 兼容 API |
| `prompts.dir` | 分类 prompt 目录（`diff_pricing.txt` 等） |

**已有库升级**（加 `signal_category` 列）：

```bash
mysql -u root -p lumor_puls < scripts/migrate_signal_category.sql
```

### Signal 分类（按 task 配置，分开执行 extractor）

| `signal_category` | prompt | `payload` 示例 |
|-------------------|--------|----------------|
| `pricing` | `diff_pricing.txt` | `model`, `old_price`, `new_price` |
| `release` | `diff_release.txt` | `product`, `version`, `breaking` |
| `protocol` / `capability` / `ecosystem` | `diff_ecosystem.txt` | `type`, `old`, `new` |

Task 仍管「去哪抓」；Signal 带 `category` + `payload`（JSON）。

**监控哪些网站**：改 MySQL，例如：

```sql
INSERT INTO tasks (id, url, `interval`, type, signal_category, enabled, created_at, updated_at)
VALUES ('deepseek_news', 'https://www.deepseek.com/news', '12h', 'browser_snapshot', 'ecosystem', 1, NOW(), NOW());

UPDATE tasks SET enabled = 0 WHERE id = 'techcrunch_ai';
DELETE FROM tasks WHERE id = 'old_task';
```

也可直接编辑 `scripts/seed_tasks.sql` 后重新导入（`ON DUPLICATE KEY UPDATE`）。

## 运行

```bash
go mod tidy
go run . 
```

浏览器打开 **http://localhost:8090/**（端口以 `config.json` 为准）可查看 Tasks / Signals 简易面板。

手动跑一次任务（建基线或触发 diff）：

```bash
go run . -run openai_pricing
```

**正常耗时**：已完成 `agent-browser install` 时，单次任务一般 **30 秒～2 分钟**（首次只建基线，不调 LLM）。超过 5 分钟多半卡在浏览器，按 `Ctrl+C` 中断后检查上面第 2 步。

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/tasks` | 任务列表（含 `lastRunAt`、`lastError`） |
| GET | `/tasks/:id` | 单个任务 |
| POST | `/tasks` | 新增任务 |
| PUT | `/tasks/:id` | 更新 url / interval / enabled 等 |
| DELETE | `/tasks/:id` | 删除任务 |
| POST | `/tasks/:id/run` | 立即执行（与调度共用浏览器锁，串行） |
| GET | `/signals` | 信号列表 |
| GET | `/signals?category=pricing&task_id=openai_pricing&limit=20` | 按分类过滤 |
| GET | `/signals?type=pricing_change` | 按细类型过滤（兼容） |

新增任务示例：

```json
POST /tasks
{
  "id": "anthropic_pricing",
  "url": "https://www.anthropic.com/pricing",
  "interval": "12h",
  "signalCategory": "pricing",
  "enabled": true
}
```

调度器对到期任务**串行**执行，日志含 `pipeline: task=xxx step=...` 便于排查。

## 流程

1. 首次运行：只保存 **baseline** snapshot，不产生 signal
2. 再次运行：若 `content_hash` 不变则跳过；否则按 `signal_category` 调用对应 extractor，写入带 `payload` 的 Signal
3. 调度器按 `interval`（如 `6h`）与 `last_run_at` 判断是否 due
