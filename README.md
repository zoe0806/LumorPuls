# LumorPuls

单节点 AI 生态**变化监控**：定时用 `agent-browser` 抓页面 → 与上次快照比对 → LLM 生成 Signal → MySQL 存储 → HTTP 输出。

## 前置

1. **MySQL**：创建库（或执行 `scripts/init_db.sql`）
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
   若必须用 `install`，需可访问 `googlechromelabs.github.io` 或使用代理后再执行 `agent-browser install`。
3. **LLM**：在 `config.json` 填 `llm.apiKey`，或环境变量 `LUMOR_LLM_API_KEY`
4. **`browser.waitNetworkIdle`**：建议保持 `false`。OpenAI 等 SPA 用 `networkidle` 可能一直等不到结束。

## 配置

编辑项目根目录 `config.json`：

- `mysqlDsn`：MySQL 连接串
- `seedTasks`：首次启动自动写入 `tasks` 表
- `scheduler.enabled`：是否后台定时跑任务

## 运行

```bash
go mod tidy
go run . 
```

手动跑一次任务（建基线或触发 diff）：

```bash
go run . -run openai_pricing
```

**正常耗时**：已完成 `agent-browser install` 时，单次任务一般 **30 秒～2 分钟**（首次只建基线，不调 LLM）。超过 5 分钟多半卡在浏览器，按 `Ctrl+C` 中断后检查上面第 2 步。

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/tasks` | 任务列表 |
| POST | `/tasks/:id/run` | 立即执行某任务 |
| GET | `/signals` | 信号列表 |
| GET | `/signals?type=pricing_change&task_id=openai_pricing&limit=20` | 过滤 |

## 流程

1. 首次运行：只保存 **baseline** snapshot，不产生 signal
2. 再次运行：若 `content_hash` 不变则跳过；否则存新 snapshot 并调用 LLM diff
3. 调度器按 `interval`（如 `6h`）与 `last_run_at` 判断是否 due
