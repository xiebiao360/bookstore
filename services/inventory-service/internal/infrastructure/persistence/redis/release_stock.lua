-- release_stock.lua
-- 库存释放Lua脚本（原子操作）
--
-- 教学要点：
-- 1. 释放场景
--    - 订单取消
--    - 支付失败
--    - 支付超时
--
-- 2. 幂等性控制
--    - 检查订单是否已释放
--    - 防止重复释放导致库存虚增
--
-- KEYS[1]: 库存键（stock:book_id）
-- ARGV[1]: 释放数量
-- ARGV[2]: 订单ID
--
-- 返回值：
-- 0: 失败
-- 1: 释放成功
-- 2: 重复释放（幂等性）

local stock_key = KEYS[1]
local quantity = tonumber(ARGV[1])
local order_id = ARGV[2]

-- 幂等性检查（订单是否已释放）
local release_record_key = "release:" .. stock_key .. ":" .. order_id
local is_released = redis.call('EXISTS', release_record_key)

if is_released == 1 then
    -- 订单已释放，返回2（幂等性）
    return 2
end

-- 检查订单是否已扣减
local deduct_record_key = "deduct:" .. stock_key .. ":" .. order_id
local is_deducted = redis.call('EXISTS', deduct_record_key)

if is_deducted == 0 then
    -- 订单未扣减，无需释放
    return 0
end

-- 增加库存（释放）
redis.call('INCRBY', stock_key, quantity)

-- 删除扣减记录
redis.call('DEL', deduct_record_key)

-- 记录已释放（有效期1小时）
redis.call('SETEX', release_record_key, 3600, '1')

-- 释放成功，返回1
return 1
