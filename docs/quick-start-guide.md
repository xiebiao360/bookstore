# 图书商城微服务 - 一键启动指南

> 本指南介绍如何一键启动Phase 2的完整微服务架构系统

---

## 🎯 系统架构

启动后将运行以下服务：

### 基础设施（4个）
1. **MySQL 8.0** - 关系型数据库（端口3306）
2. **Redis 7.x** - 缓存与会话存储（端口6379）
3. **RabbitMQ 3.12** - 消息队列（端口5672 + 管理界面15672）
4. **Jaeger** - 分布式追踪（UI端口16686）

### 微服务（6个）
1. **API Gateway** - 统一入口，HTTP→gRPC转换（端口8080）
2. **User Service** - 用户认证与管理（gRPC端口50051）
3. **Catalog Service** - 图书目录与搜索（gRPC端口50052）
4. **Inventory Service** - 库存管理（gRPC端口50053）
5. **Payment Service** - 支付处理（gRPC端口50054）
6. **Order Service** - 订单管理（gRPC端口50055）

---

## 🚀 一键启动

### 方法1: 使用Make命令（推荐）

```bash
# 在项目根目录执行
make start
```

**执行流程**：
1. 检查依赖（Docker、Go、nc）
2. 启动基础设施（MySQL、Redis、RabbitMQ、Jaeger）
3. 编译所有微服务
4. 启动所有微服务（后台运行）
5. 健康检查
6. 显示访问信息

**预计耗时**: 约30-60秒

### 方法2: 直接执行脚本

```bash
# 在项目根目录执行
./scripts/start-all.sh
```

---

## 🛑 停止服务

### 停止所有服务

```bash
make stop
# 或
./scripts/stop-all.sh
```

### 仅停止微服务（保留基础设施）

```bash
# 手动停止各个微服务进程
kill $(cat logs/*.pid)
```

### 仅停止基础设施

```bash
docker compose down
```

---

## 🔄 重启服务

```bash
make restart
# 或
./scripts/restart-all.sh
```

---

## 📊 访问地址汇总

### 基础设施UI

| 服务 | 访问地址 | 凭证 |
|-----|---------|------|
| **phpMyAdmin** | http://localhost:8081 | 用户: root<br>密码: root123 |
| **RabbitMQ管理** | http://localhost:15672 | 用户: admin<br>密码: admin123 |
| **Jaeger UI** | http://localhost:16686 | 无需认证 |

### API Gateway

| 端点 | 地址 | 说明 |
|-----|------|------|
| **健康检查** | http://localhost:8080/health | 查看服务状态 |
| **API文档** | http://localhost:8080/swagger | Swagger UI（如已集成） |

---

## 📝 查看日志

### 实时查看所有日志

```bash
make logs
# 或
tail -f logs/*.log
```

### 查看特定服务日志

```bash
# 查看API Gateway日志
tail -f logs/api-gateway.log

# 查看订单服务日志
tail -f logs/order-service.log

# 查看用户服务日志
tail -f logs/user-service.log
```

### 查看Docker容器日志

```bash
# 查看所有容器日志
docker compose logs -f

# 查看特定容器日志
docker compose logs -f mysql
docker compose logs -f rabbitmq
docker compose logs -f jaeger
```

---

## 🧪 测试服务可用性

### 1. 测试API Gateway

```bash
# 健康检查
curl http://localhost:8080/health

# 预期响应
# {"status":"ok","timestamp":"2025-11-06T10:00:00Z"}
```

### 2. 测试用户注册

```bash
curl -X POST http://localhost:8080/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "nickname": "测试用户"
  }'
```

### 3. 测试用户登录

```bash
curl -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

### 4. 查看Jaeger追踪

1. 访问 http://localhost:16686
2. 选择服务（如`api-gateway`、`order-service`）
3. 点击"Find Traces"查看请求链路

### 5. 查看RabbitMQ队列

1. 访问 http://localhost:15672
2. 登录（admin/admin123）
3. 查看Exchanges、Queues、Messages

---

## 🔧 常见问题

### Q1: 端口被占用怎么办？

**问题**：启动时提示端口已被占用

**解决方案**：
```bash
# 查看端口占用情况
lsof -i :8080     # API Gateway
lsof -i :3306     # MySQL
lsof -i :5672     # RabbitMQ

# 停止占用端口的进程
kill -9 <PID>

# 或修改docker-compose.yml中的端口映射
```

### Q2: 服务启动失败怎么办？

**问题**：某个微服务启动失败

**解决方案**：
```bash
# 1. 查看日志
tail -f logs/order-service.log

# 2. 检查进程状态
ps aux | grep order-service

# 3. 检查端口是否被占用
lsof -i :50055

# 4. 手动启动服务调试
cd services/order-service
go run cmd/main.go
```

### Q3: 数据库连接失败怎么办？

**问题**：服务日志显示数据库连接失败

**解决方案**：
```bash
# 1. 检查MySQL是否启动
docker ps | grep mysql

# 2. 检查MySQL健康状态
docker compose ps

# 3. 测试数据库连接
mysql -h 127.0.0.1 -P 3306 -u bookstore -pbookstore123

# 4. 重启MySQL容器
docker compose restart mysql
```

### Q4: RabbitMQ连接失败怎么办？

**问题**：消息队列功能不可用

**解决方案**：
```bash
# 1. 检查RabbitMQ是否启动
docker ps | grep rabbitmq

# 2. 查看RabbitMQ日志
docker compose logs rabbitmq

# 3. 重启RabbitMQ
docker compose restart rabbitmq

# 4. 访问管理界面检查
open http://localhost:15672
```

### Q5: 如何清理所有数据重新开始？

```bash
# 1. 停止所有服务
make stop

# 2. 清理Docker volumes（会删除所有数据）
docker compose down -v

# 3. 清理编译产物和日志
make clean

# 4. 重新启动
make start
```

---

## 📈 性能监控

### 使用Jaeger查看请求追踪

1. 访问 http://localhost:16686
2. 选择服务和时间范围
3. 查看调用链路和耗时分布
4. 定位性能瓶颈

**示例查询**：
- 查找耗时>1s的请求：Duration > 1s
- 查找失败的请求：Tags: error=true
- 查找特定用户的请求：Tags: user_id=123

### 查看Prometheus指标（如已启动）

```bash
# 查看API Gateway metrics端点
curl http://localhost:9090/metrics

# 常见指标：
# - http_requests_total - 请求总数
# - http_request_duration_seconds - 请求耗时
# - orders_created_total - 订单创建总数
# - circuit_breaker_state - 熔断器状态
```

---

## 🛠️ 开发模式

### 单独启动某个服务（开发调试）

```bash
# 1. 先启动基础设施
docker compose up -d

# 2. 启动依赖的服务（如user-service）
cd services/user-service
go run cmd/main.go

# 3. 在另一个终端启动要调试的服务
cd services/order-service
go run cmd/main.go
```

### 使用热重载（推荐安装air）

```bash
# 安装air
go install github.com/cosmtrek/air@latest

# 在服务目录下启动
cd services/order-service
air

# air会监听文件变化自动重新编译
```

---

## 📚 下一步

启动成功后，可以：

1. **学习微服务通信**：查看gRPC接口定义（proto文件）
2. **体验分布式追踪**：在Jaeger UI查看完整的请求链路
3. **测试消息队列**：查看RabbitMQ中的消息流转
4. **性能优化**：使用Jaeger定位慢请求瓶颈
5. **故障模拟**：关闭某个服务，观察熔断器行为

**完整学习路径**：参考 [ROADMAP.md](../ROADMAP.md)

---

## 🎓 教学价值

通过一键启动，你将体验到：

✅ **完整的微服务生态**：6个服务 + 4个基础设施  
✅ **分布式追踪**：Jaeger可视化调用链路  
✅ **消息队列**：RabbitMQ异步解耦  
✅ **服务编排**：Docker Compose一键管理  
✅ **可观测性**：日志、追踪、监控三位一体  

**这是一个真实的生产级微服务系统的缩影！**

---

**有问题？** 查看项目文档或提Issue: https://github.com/xiebiao/bookstore/issues
