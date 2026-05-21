# Lumor Puls (MVP)

单节点 AI 生态**变化监控**：定时用 `agent-browser` 抓页面 → 与上次快照比对 → LLM 生成 Signal → MySQL 存储 → HTTP 输出。

## 前置

1. **MySQL**：创建库（或执行 `scripts/init_db.sql`）
2. **agent-browser**：`npm install -g agent-browser && agent-browser install`
3. **LLM**：在 `config.json` 填 `llm.apiKey`，或环境变量 `LUMOR_LLM_API_KEY`

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
