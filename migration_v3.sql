-- chat-token 好友表重构：万能外键 → 三张独立关联表
-- 执行: psql -U qiuming -d 12fzsj -f migration_v3.sql

-- ══════════════════════════════════════════
-- 1. 新建三张关联表
-- ══════════════════════════════════════════

-- 人↔人 好友关系
CREATE TABLE IF NOT EXISTS chat.contacts (
    user_id     TEXT NOT NULL,
    contact_id  TEXT NOT NULL,
    status      TEXT DEFAULT 'accepted',
    category    TEXT DEFAULT '日常',
    org_id      UUID,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, contact_id)
);

-- 人↔Agent 关联
CREATE TABLE IF NOT EXISTS chat.user_agents (
    user_id     TEXT NOT NULL,
    agent_id    TEXT NOT NULL REFERENCES chat.agents(bot_id) ON DELETE CASCADE,
    category    TEXT DEFAULT '日常',
    org_id      UUID,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, agent_id)
);

-- 人↔主机 关联
CREATE TABLE IF NOT EXISTS chat.user_devices (
    user_id     TEXT NOT NULL,
    device_id   TEXT NOT NULL REFERENCES chat.devices(id) ON DELETE CASCADE,
    org_id      UUID,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, device_id)
);

-- ══════════════════════════════════════════
-- 2. 从旧表迁移数据到新表
-- ══════════════════════════════════════════

-- 人↔人：user_type = 'human' 或无 user_type 且不在 agent/device 中
INSERT INTO chat.contacts (user_id, contact_id, status, category, created_at)
SELECT DISTINCT f.user_id, f.friend_id, COALESCE(f.status,'accepted'),
       COALESCE(f.category,'日常'), f.created_at
FROM chat.friends f
WHERE f.user_type = 'human'
   OR (f.user_type IS NULL
       AND f.friend_id NOT IN (SELECT bot_id FROM chat.agents)
       AND f.friend_id NOT IN (SELECT id FROM chat.devices))
ON CONFLICT DO NOTHING;

-- 人↔Agent：user_type = 'agent' 或 friend_id 在 agents 中
INSERT INTO chat.user_agents (user_id, agent_id, category, created_at)
SELECT DISTINCT f.user_id, f.friend_id,
       COALESCE(f.category,'日常'), f.created_at
FROM chat.friends f
WHERE f.user_type = 'agent'
   OR f.friend_id IN (SELECT bot_id FROM chat.agents)
ON CONFLICT DO NOTHING;

-- 人↔主机：user_type = 'device' 或 friend_id 在 devices 中
INSERT INTO chat.user_devices (user_id, device_id, created_at)
SELECT DISTINCT f.user_id, f.friend_id, f.created_at
FROM chat.friends f
WHERE f.user_type = 'device'
   OR f.friend_id IN (SELECT id FROM chat.devices)
ON CONFLICT DO NOTHING;

-- ══════════════════════════════════════════
-- 3. 重命名旧表（保留备份）
-- ══════════════════════════════════════════
ALTER TABLE chat.friends RENAME TO chat.friends_old;

-- 兼容视图（过渡期：读新表 + 旧表 UNION）
CREATE OR REPLACE VIEW chat.friends AS
SELECT user_id, contact_id AS friend_id, status, category, 'human' AS user_type, created_at FROM chat.contacts
UNION ALL
SELECT ua.user_id, ua.agent_id AS friend_id, 'accepted' AS status, ua.category, 'agent' AS user_type, ua.created_at FROM chat.user_agents ua
UNION ALL
SELECT ud.user_id, ud.device_id AS friend_id, 'accepted' AS status, '日常' AS category, 'device' AS user_type, ud.created_at FROM chat.user_devices ud
UNION ALL
SELECT user_id, friend_id, status, COALESCE(category,'日常'), COALESCE(user_type,'human'), created_at FROM chat.friends_old
WHERE (user_id, friend_id) NOT IN (
    SELECT user_id, contact_id FROM chat.contacts
    UNION ALL SELECT user_id, agent_id FROM chat.user_agents
    UNION ALL SELECT user_id, device_id FROM chat.user_devices
);
