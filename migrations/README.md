# 数据库迁移说明

## 📋 迁移文件列表

### MySQL
```
mysql/
├── 00-init-db.sql                          # ❗️初始化脚本（包含后期添加的字段）
├── 02-add-agent-config-if-missing.sql     # ✅ 兼容性迁移（检查并添加缺失字段）
└── 03-cleanup-unreferenced-models.sql     # 🧹 清理未引用模型（维护脚本）
```

### ParadeDB/PostgreSQL
```
paradedb/
├── 01-migrate-to-paradedb.sql             # ❗️ParadeDB 初始化
├── 02-add-agent-config-if-missing.sql     # ✅ 兼容性迁移（检查并添加缺失字段）
└── 03-cleanup-unreferenced-models.sql     # 🧹 清理未引用模型（维护脚本）
```

## ⚠️ 重要说明

### 关于 `agent_config` 等字段

你可能会注意到 `agent_config`、`context_config`、`agent_steps` 这些字段**同时出现在**：
- ✅ 初始化脚本中（`00-init-db.sql`）
- ✅ 迁移脚本中（`02-add-agent-config-if-missing.sql`）

**这不是错误！** 这是为了兼容两种场景：

#### 场景1: 全新安装 🆕
运行 `00-init-db.sql` 时，会直接创建包含所有字段的表（包括 `agent_config`）。
当运行 `02-add-agent-config-if-missing.sql` 时，会检测到字段已存在，跳过添加。

#### 场景2: 旧版本升级 ⬆️
如果你的数据库是从旧版本（不包含 `agent_config` 的版本）升级：
- `00-init-db.sql` 早已运行过（但那时还没有 `agent_config` 字段）
- 运行 `02-add-agent-config-if-missing.sql` 时，会检测到字段不存在，然后添加

### 为什么会这样？

根据 `AGENT_CONFIG_TENANT_REFACTORING.md` 文档，`agent_config` 是后来添加的功能。理论上：

**❌ 不应该做的（但已经做了）：**
```sql
-- 00-init-db.sql 被修改了，添加了后来才加的字段
CREATE TABLE tenants (
    agent_config JSON DEFAULT NULL,  -- 这个字段是后来加的！
    ...
);
```

**✅ 应该做的：**
```sql
-- 00-init-db.sql（保持不变）
CREATE TABLE tenants (
    -- 没有 agent_config
    ...
);

-- 02-add-agent-config.sql（新文件）
ALTER TABLE tenants ADD COLUMN agent_config JSON DEFAULT NULL;
```

但由于初始化脚本已经被修改，我们创建了一个**智能的兼容性迁移脚本** `02-add-agent-config-if-missing.sql`，它：
- ✅ 检查字段是否存在
- ✅ 只在字段不存在时才添加
- ✅ 对新旧数据库都安全

## 🚀 使用方法

### 方法1: 全新部署

```bash
# 直接运行所有迁移
./scripts/migrate.sh up

# 结果：
# - 00-init-db.sql: 创建表（包含 agent_config）
# - 02-add-agent-config-if-missing.sql: 检测到字段已存在，跳过
```

### 方法2: 从旧版本升级

```bash
# 运行新的迁移
docker exec -i paradedb_container psql -U postgres -d WeKnora \
  < migrations/paradedb/02-add-agent-config-if-missing.sql

# 结果：
# - 检测到 agent_config 不存在
# - 添加 agent_config、context_config、agent_steps 字段
# - 创建 GIN 索引提高查询性能
```

### 方法3: 验证字段是否存在

#### MySQL
```sql
SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = 'WeKnora'
  AND TABLE_NAME = 'tenants'
  AND COLUMN_NAME = 'agent_config';
```

#### PostgreSQL
```sql
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name = 'tenants'
  AND column_name = 'agent_config';
```

## 📝 涉及的字段

以下字段是通过 `02-add-agent-config-if-missing.sql` 处理的：

### `tenants` 表
- `agent_config` (JSON/JSONB): 租户级别的 Agent 配置

### `sessions` 表
- `agent_config` (JSON/JSONB): 会话级别的 Agent 配置
- `context_config` (JSON/JSONB): LLM 上下文管理配置

### `messages` 表
- `agent_steps` (JSON/JSONB): Agent 执行步骤（推理过程和工具调用）

## 🎯 最佳实践建议

### 对于新字段的添加

如果你需要添加新字段，**不要修改 `00-init-db.sql`**！应该：

1. 创建新的迁移文件：`03-your-feature.sql`
2. 使用 `ALTER TABLE` 添加字段
3. 使用 `IF NOT EXISTS` 确保幂等性

示例：
```sql
-- 03-add-new-feature.sql
ALTER TABLE tenants 
ADD COLUMN IF NOT EXISTS new_field VARCHAR(255) DEFAULT NULL;
```

### 对于现有字段的修改

```sql
-- 04-modify-field.sql
ALTER TABLE tenants 
MODIFY COLUMN existing_field TEXT NULL;
```

## 📚 更多信息

详细的迁移指南请参考：[MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md)

## 🧹 维护脚本

### 03-cleanup-unreferenced-models.sql

**用途**: 清理数据库中未被任何知识库引用的模型

**清理条件**（必须同时满足）：
- ❌ 未被 `knowledge_bases` 引用（embedding/summary/rerank/vlm 模型）
- ❌ 未被 `knowledges` 引用（embedding 模型）
- ❌ 非默认模型（`is_default = FALSE`）
- ❌ 非系统模型（`tenant_id != 0`）
- ❌ 未软删除（`deleted_at IS NULL`）

**使用步骤**：
1. 📊 运行 DRY RUN 查询查看将被删除的模型
2. 💾 备份数据库
3. 🔄 选择软删除（推荐）或硬删除
4. ✅ 验证结果

**详细说明**: 请查看 [CLEANUP_GUIDE.md](./CLEANUP_GUIDE.md)

**⚠️ 重要**：
- 软删除可以回滚，硬删除不可逆
- 建议先在测试环境验证
- 可设置定时任务自动清理

## 🔗 相关文档

- `AGENT_CONFIG_TENANT_REFACTORING.md` - Agent 配置重构说明
- `CONTEXT_MANAGER_IMPLEMENTATION.md` - 上下文管理器实现
- `MIGRATION_GUIDE.md` - 完整的迁移指南
- `CLEANUP_GUIDE.md` - 模型清理指南 🆕

---

**创建时间**: 2025-11-03  
**问题发现**: Agent Config 字段既在初始化脚本中，又应该在增量迁移中  
**解决方案**: 创建智能兼容性迁移脚本，同时支持新旧数据库

**更新时间**: 2025-11-03  
**新增功能**: 添加未引用模型清理脚本，支持软删除和硬删除

