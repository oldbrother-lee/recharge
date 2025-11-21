-- 迁移拉单源权限到平台账号权限
-- 删除旧的拉单源权限
DELETE FROM permissions WHERE code IN (
    'pullsource',
    'pullsource:list',
    'pullsource:add',
    'pullsource:edit',
    'pullsource:delete',
    'pullsource:variant:list',
    'pullsource:variant:add',
    'pullsource:variant:edit',
    'pullsource:variant:delete'
);

-- 添加新的平台账号权限
INSERT INTO permissions (code, name, type, path, component, icon, description, `show`, enable, `order`, created_at, updated_at) VALUES
-- 平台账号管理菜单
('platform:account', '平台账号管理', 'MENU', '/platform', '/src/views/platform/index.vue', 'mdi:account-multiple', '平台账号管理菜单', 1, 1, 9, NOW(), NOW()),

-- 平台账号基础权限
('platform:account:list', '查看平台账号', 'BUTTON', NULL, NULL, NULL, '查看平台账号列表权限', 1, 1, 1, NOW(), NOW()),
('platform:account:add', '新增平台账号', 'BUTTON', NULL, NULL, NULL, '新增平台账号权限', 1, 1, 2, NOW(), NOW()),
('platform:account:edit', '编辑平台账号', 'BUTTON', NULL, NULL, NULL, '编辑平台账号权限', 1, 1, 3, NOW(), NOW()),
('platform:account:delete', '删除平台账号', 'BUTTON', NULL, NULL, NULL, '删除平台账号权限', 1, 1, 4, NOW(), NOW()),

-- 平台账号变体权限
('platform:account:variant:list', '查看变体配置', 'BUTTON', NULL, NULL, NULL, '查看平台账号变体配置权限', 1, 1, 5, NOW(), NOW()),
('platform:account:variant:add', '新增变体配置', 'BUTTON', NULL, NULL, NULL, '新增平台账号变体配置权限', 1, 1, 6, NOW(), NOW()),
('platform:account:variant:edit', '编辑变体配置', 'BUTTON', NULL, NULL, NULL, '编辑平台账号变体配置权限', 1, 1, 7, NOW(), NOW()),
('platform:account:variant:delete', '删除变体配置', 'BUTTON', NULL, NULL, NULL, '删除平台账号变体配置权限', 1, 1, 8, NOW(), NOW());