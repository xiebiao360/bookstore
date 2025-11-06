-- restock_inventory.lua
-- 库存补充Lua脚本（原子操作）
--
-- 教学要点：
-- 1. 补货场景
--    - 管理员补货
--    - 退货入库
--
-- KEYS[1]: 库存键（stock:book_id）
-- ARGV[1]: 补充数量
--
-- 返回值：补充后的库存数量

local stock_key = KEYS[1]
local quantity = tonumber(ARGV[1])

-- 增加库存
local new_stock = redis.call('INCRBY', stock_key, quantity)

-- 返回新库存
return new_stock
