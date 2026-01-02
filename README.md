# CEX-Study

这是一个 **CEX 出入金 + 内部账本** 的原型系统：链上只负责资金流入/流出，用户余额由数据库账本维护

## 实现了什么

- **充值（Deposit）**：监听指定 ERC20（示例：`USDC_TOKEN`）的 `Transfer`，若 `to == EXCHANGE_ADDRESS` 写入 `deposits(PENDING)`；达到确认数后置状态 `CONFIRMED` 并入账（`ledger_entries` + `balances`）。
- **提现（Withdraw）**：`POST /api/v1/withdraw` 创建提现单 → 先冻结余额（`WITHDRAW_HOLD`）→ 热钱包发 ERC20 转账 → 保存 `tx_hash` 并置 `SENT`；索引器监听热钱包转出，匹配 `tx_hash` 后置 `CONFIRMED` 并写入 `WITHDRAW_FINAL`。
- **Reorg 处理**：本地保存 `blocks(number,hash,parent_hash)`；检测到 reorg 后暂停索引并派发回滚任务，回滚受影响区块内充值并做反向记账。
	- 使用 Redis 实现 “轻量状态 + 事件派发”：
		- `indexer:block_height`：索引进度 checkpoint（重启可续跑）
		- `indexer:paused`：reorg 时暂停/恢复索引
		- `reorg_jobs`（Redis Stream）：发布/消费 reorg 起点区块号，`DepositWorkflow` 回滚后 ack
	- 备注：Redis 也应该承担分布式锁的职责（例如索引器/workflow单实例运行、热钱包 nonce 管理等），目前尚未实现。

## 关键数据表

- `deposits`：充值流水（`PENDING/CONFIRMED/REVERTED`）
- `withdraws`：提现单（`REQUESTED/SENT/CONFIRMED/FAILED`）
- `ledger_entries`：账本分录（审计流水）
- `balances`：余额快照（读优化）
- `deposit_addresses`：用户充值地址映射（用 from 地址识别 user）
- `blocks`：区块 hash 记录（reorg 对比）

## 快速启动

1) 准备 Postgres/Redis（Redis 用于索引进度/暂停标记/reorg job）

2) 配置 `.env`（关键项）

- `ETH_RPC_URL`
- `USDC_TOKEN`
- `EXCHANGE_ADDRESS`（收款地址/热钱包地址）
- `PRIVATE_KEY`（热钱包私钥，用于提现转账）
- `DB_*` / `REDIS_*`

3) 运行

```bash
go run ./cmd/main.go
```