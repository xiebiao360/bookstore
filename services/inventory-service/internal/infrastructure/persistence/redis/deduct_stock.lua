-- deduct_stock.lua
-- 库存扣减Lua脚本（原子操作）
--
-- 教学要点：
-- 1. 为什么使用Lua脚本？
--    - Redis单线程执行，Lua脚本原子性
--    - 避免WATCH/MULTI/EXEC的复杂性
--    - 性能高（减少网络往返次数）
--
-- 2. 脚本逻辑
--    - 检查库存是否充足
--    - 原子扣减库存
--    - 返回结果（0=库存不足，1=成功）
--
-- 3. 幂等性控制
--    - 使用订单ID作为去重键
--    - 防止同一订单重复扣减
--
-- KEYS[1]: 库存键（stock:book_id）
-- ARGV[1]: 扣减数量
-- ARGV[2]: 订单ID（用于幂等性控制）
--
-- 返回值：
-- 0: 库存不足
-- 1: 扣减成功
-- 2: 重复扣减（幂等性）

-- 获取当前库存
local stock_key = KEYS[1]
local quantity = tonumber(ARGV[1])
local order_id = ARGV[2]

-- 幂等性检查（订单是否已处理）
local deduct_record_key = "deduct:" .. stock_key .. ":" .. order_id
local is_deducted = redis.call('EXISTS', deduct_record_key)

if is_deducted == 1 then
    -- 订单已扣减，返回2（幂等性）
    return 2
end

-- 获取当前库存
local current_stock = tonumber(redis.call('GET', stock_key) or 0)

-- 检查库存是否充足
if current_stock < quantity then
    -- 库存不足，返回0
    return 0
end

-- 扣减库存
redis.call('DECRBY', stock_key, quantity)

-- 记录已扣减（有效期1小时，防止内存泄漏）
redis.call('SETEX', deduct_record_key, 3600, '1')

-- 扣减成功，返回1
return 1
