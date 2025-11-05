# Day 20: pprof 性能分析工具使用指南

> **教学目标**：掌握Go语言性能分析的核心工具pprof，学会定位CPU瓶颈、内存泄漏和goroutine泄漏

---

## 📋 目录

1. [pprof 简介](#pprof-简介)
2. [集成方式](#集成方式)
3. [性能分析类型](#性能分析类型)
4. [实战教程](#实战教程)
5. [可视化分析](#可视化分析)
6. [生产环境最佳实践](#生产环境最佳实践)
7. [常见问题排查](#常见问题排查)

---

## pprof 简介

### 什么是 pprof？

`pprof` 是 Go 官方提供的性能分析工具，可以：
- **CPU Profiling**：找出最耗 CPU 的函数
- **Memory Profiling**：分析内存分配和泄漏
- **Goroutine Profiling**：检测 goroutine 泄漏
- **Block Profiling**：分析阻塞操作（锁竞争、通道操作）
- **Mutex Profiling**：分析互斥锁争用

### 为什么需要性能分析？

```go
// ❌ 没有性能分析时的开发流程
开发功能 → 上线 → 发现慢 → 盲目猜测瓶颈 → 随机优化 → 可能更慢

// ✅ 有性能分析时的开发流程
开发功能 → 压测 → pprof分析 → 精确定位瓶颈 → 针对性优化 → 验证效果
```

---

## 集成方式

### 1. 导入 pprof 包

```go
import _ "net/http/pprof"
```

只需一行代码，pprof 会自动注册以下路由到 `http.DefaultServeMux`：
- `/debug/pprof/` - 主页（所有分析类型）
- `/debug/pprof/profile` - CPU 分析
- `/debug/pprof/heap` - 内存分配分析
- `/debug/pprof/goroutine` - goroutine 分析
- `/debug/pprof/block` - 阻塞分析
- `/debug/pprof/mutex` - 互斥锁分析

### 2. 启动 pprof HTTP 服务器

**方式 1：独立端口（推荐）**

```go
// 在 main.go 中启动独立的 pprof 服务器
go func() {
    log.Println(http.ListenAndServe(":6060", nil))
}()
```

**优点**：
- 业务服务和性能分析服务隔离
- 生产环境可以通过防火墙限制 6060 端口访问
- 避免性能分析影响业务请求

**方式 2：与业务服务共享端口（不推荐生产环境）**

```go
// 将 pprof 路由注册到 Gin
router := gin.Default()
router.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))
```

**缺点**：
- pprof 端点暴露在公网（安全风险）
- 性能分析可能影响业务请求

---

## 性能分析类型

### 1. CPU Profiling（CPU 分析）

**用途**：找出哪些函数消耗了最多的 CPU 时间

**采集方法**：

```bash
# 方法1：命令行采集（采集30秒）
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 方法2：下载profile文件后分析
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof
```

**典型场景**：
- API 响应慢，不知道慢在哪里
- CPU 使用率高，需要找出热点函数
- 压测后发现 QPS 上不去

**教学示例**：

假设我们的图书列表接口很慢，通过 CPU profiling 发现：

```
(pprof) top10
Showing nodes accounting for 3.50s, 70.00% of 5.00s total
Dropped 45 nodes (cum <= 0.025s)
Showing top 10 nodes out of 58
      flat  flat%   sum%        cum   cum%
     1.20s 24.00% 24.00%      1.20s 24.00%  runtime.mallocgc
     0.80s 16.00% 40.00%      0.80s 16.00%  encoding/json.Marshal
     0.60s 12.00% 52.00%      0.60s 12.00%  database/sql.(*Rows).Next
     0.40s  8.00% 60.00%      2.50s 50.00%  bookstore/internal/application/book.(*ListBooksUseCase).Execute
```

**分析结果**：
- `json.Marshal` 占用 16% CPU → 可能是序列化了不必要的字段
- `sql.Rows.Next` 占用 12% CPU → 可能是查询了太多数据

**优化方向**：
- 减少返回字段（不返回 `Description` 等大字段）
- 添加分页限制（防止一次查询 10000 条数据）

---

### 2. Memory Profiling（内存分析）

**用途**：分析内存分配情况，找出内存泄漏

**采集方法**：

```bash
# 方法1：实时内存分配分析
go tool pprof http://localhost:6060/debug/pprof/heap

# 方法2：查看内存分配速率（allocs）
go tool pprof http://localhost:6060/debug/pprof/allocs

# 方法3：下载heap文件
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**heap vs allocs 的区别**：
- `heap`：当前内存中存活的对象（已减去 GC 回收的）
- `allocs`：累计分配的所有对象（包括已回收的）

**典型场景**：
- 内存使用持续增长（内存泄漏）
- GC 频繁触发（分配速率过高）
- OOM（Out of Memory）问题排查

**教学示例**：

```bash
$ go tool pprof http://localhost:6060/debug/pprof/heap
(pprof) top10
Showing nodes accounting for 512.51MB, 90.15% of 568.45MB total
Dropped 20 nodes (cum <= 2.84MB)
Showing top 10 nodes out of 45
      flat  flat%   sum%        cum   cum%
  200.50MB 35.27% 35.27%   200.50MB 35.27%  bookstore/internal/domain/book.(*Book).MarshalJSON
  150.20MB 26.42% 61.69%   150.20MB 26.42%  github.com/gin-gonic/gin.(*Context).JSON
```

**分析结果**：
- `Book.MarshalJSON` 占用 200MB → 可能是缓存了太多图书对象
- 建议：限制缓存大小，使用 LRU 淘汰策略

---

### 3. Goroutine Profiling（协程分析）

**用途**：检测 goroutine 泄漏

**采集方法**：

```bash
# 方法1：查看当前goroutine数量和调用栈
go tool pprof http://localhost:6060/debug/pprof/goroutine

# 方法2：浏览器查看
open http://localhost:6060/debug/pprof/goroutine?debug=1
```

**典型场景**：
- goroutine 数量持续增长（泄漏）
- 服务运行一段时间后变慢
- 怀疑有死锁或永久阻塞的 goroutine

**教学示例**：

正常情况下，goroutine 数量应该稳定：

```
$ curl http://localhost:6060/debug/pprof/goroutine?debug=1 | head -20
goroutine profile: total 25
10 @ 0x43a385 0x43a230 0x409321 0x408f05 0x78c2c7 0x465a41
#   10 goroutines 等待在 select 操作上（正常）

5 @ 0x43a385 0x44b3c5 0x44b39e 0x44af52 0x4e8b43 0x465a41
#   5 goroutines 等待在 channel 接收上（正常）
```

异常情况（goroutine 泄漏）：

```
goroutine profile: total 10025  # 😱 goroutine数量异常！
10000 @ 0x43a385 0x409321 0x78c2c7 0x465a41
#   bookstore/internal/infrastructure/cache.(*RedisCache).watchExpiration
#   可能原因：每次调用都创建了新goroutine，但没有退出机制
```

**修复方法**：

```go
// ❌ 错误：无限创建goroutine
func (c *Cache) Get(key string) {
    go func() {
        // 没有退出条件，goroutine会一直运行
        for {
            time.Sleep(time.Second)
            c.refresh(key)
        }
    }()
}

// ✅ 正确：使用context控制goroutine生命周期
func (c *Cache) Get(ctx context.Context, key string) {
    go func() {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                c.refresh(key)
            case <-ctx.Done(): // 当context取消时退出
                return
            }
        }
    }()
}
```

---

### 4. Block Profiling（阻塞分析）

**用途**：分析哪些操作导致 goroutine 阻塞（锁、通道、I/O）

**启用方式**：

```go
// 在main.go中启用block profiling
import "runtime"

func main() {
    runtime.SetBlockProfileRate(1) // 启用阻塞分析
    // ...
}
```

**采集方法**：

```bash
go tool pprof http://localhost:6060/debug/pprof/block
```

**典型场景**：
- 并发性能差（大量 goroutine 等待锁）
- channel 操作频繁阻塞
- 数据库连接池耗尽

---

### 5. Mutex Profiling（互斥锁分析）

**用途**：分析互斥锁的争用情况

**启用方式**：

```go
import "runtime"

func main() {
    runtime.SetMutexProfileFraction(1) // 启用互斥锁分析
    // ...
}
```

**采集方法**：

```bash
go tool pprof http://localhost:6060/debug/pprof/mutex
```

**典型场景**：
- 高并发下性能下降
- 怀疑锁竞争严重

---

## 实战教程

### 场景 1：定位图书列表接口慢的原因

**步骤 1：启动服务**

```bash
make run
```

**步骤 2：压测接口**

```bash
# 安装wrk（压测工具）
# macOS: brew install wrk
# Ubuntu: sudo apt install wrk

# 压测图书列表接口（100并发，持续60秒）
wrk -t10 -c100 -d60s http://localhost:8080/api/v1/books
```

**步骤 3：采集 CPU profile（压测期间）**

```bash
# 新开一个终端，采集30秒的CPU数据
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 等待30秒后，pprof会自动进入交互模式
```

**步骤 4：分析 CPU 热点**

```bash
# pprof交互模式下的常用命令
(pprof) top10          # 显示CPU占用最高的10个函数
(pprof) list BookList  # 显示BookList函数的源码和CPU占用
(pprof) web            # 生成调用图（需要安装graphviz）
(pprof) pdf            # 导出PDF报告
```

**步骤 5：针对性优化**

假设发现 `json.Marshal` 占用很高，优化方案：

```go
// 优化前：返回所有字段
type BookResponse struct {
    ID          uint   `json:"id"`
    Title       string `json:"title"`
    Description string `json:"description"` // 大字段，列表不需要
    Content     string `json:"content"`     // 更大的字段
}

// 优化后：列表只返回必要字段
type BookListItem struct {
    ID     uint   `json:"id"`
    Title  string `json:"title"`
    Price  int64  `json:"price"`
}
```

---

### 场景 2：排查内存泄漏

**步骤 1：观察内存增长**

```bash
# 监控内存使用（每秒采集一次）
watch -n 1 'curl -s http://localhost:6060/debug/pprof/heap | grep "# runtime.MemStats"'
```

**步骤 2：对比不同时间点的堆快照**

```bash
# 启动服务后立即采集基线
curl http://localhost:6060/debug/pprof/heap > heap_baseline.prof

# 压测1小时后采集
curl http://localhost:6060/debug/pprof/heap > heap_after_1h.prof

# 对比两个快照
go tool pprof -base=heap_baseline.prof heap_after_1h.prof
```

**步骤 3：分析增长的对象**

```bash
(pprof) top10
# 会显示相比baseline增长最多的对象类型
```

---

### 场景 3：检测 goroutine 泄漏

**步骤 1：查看 goroutine 数量趋势**

```bash
# 查看当前goroutine数量
curl http://localhost:6060/debug/pprof/goroutine?debug=1 | head -1

# 输出示例：
# goroutine profile: total 25  # 正常
# goroutine profile: total 10025  # 异常！
```

**步骤 2：分析 goroutine 调用栈**

```bash
go tool pprof http://localhost:6060/debug/pprof/goroutine

(pprof) top10
# 会显示哪些函数创建了最多的goroutine
```

**步骤 3：定位泄漏代码**

```bash
(pprof) list 函数名
# 会显示该函数的源码和goroutine创建点
```

---

## 可视化分析

### 1. 火焰图（Flame Graph）

**安装工具**：

```bash
# 安装go-torch（已内置在新版pprof中）
go install github.com/uber/go-torch@latest

# 或使用pprof内置的web界面
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=30
```

**浏览器打开**：

```
http://localhost:8081/ui/flamegraph
```

**火焰图解读**：
- 横轴：函数占用的CPU时间比例（越宽越慢）
- 纵轴：调用栈深度
- 颜色：随机分配（无特殊含义）
- 点击可以放大查看

---

### 2. 调用图（Call Graph）

**生成方法**：

```bash
# 需要先安装graphviz
# macOS: brew install graphviz
# Ubuntu: sudo apt install graphviz

# 生成PNG图片
go tool pprof -png http://localhost:6060/debug/pprof/profile?seconds=30 > profile.png

# 或在pprof交互模式下
(pprof) web  # 自动在浏览器打开
(pprof) pdf  # 生成PDF
```

**图形解读**：
- 方框：函数
- 方框大小：CPU占用（越大越慢）
- 箭头：调用关系
- 箭头粗细：调用次数

---

### 3. pprof Web UI（推荐）

**启动方式**：

```bash
# 一键启动Web界面（最方便）
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=30
```

**功能**：
- Graph：调用图
- Flame Graph：火焰图
- Top：热点函数列表
- Source：源码级分析
- Disassemble：汇编级分析

---

## 生产环境最佳实践

### 1. 安全性

**❌ 错误做法**：
```go
// 将pprof暴露在公网
router.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))
```

**✅ 正确做法**：

```go
// 方案1：独立端口 + 防火墙限制
go func() {
    // 只监听内网IP
    log.Println(http.ListenAndServe("127.0.0.1:6060", nil))
}()

// 方案2：添加认证
pprofRouter := router.Group("/debug/pprof")
pprofRouter.Use(AdminAuthRequired()) // 自定义中间件
pprofRouter.GET("/*any", gin.WrapH(http.DefaultServeMux))

// 方案3：仅在特定环境启用
if os.Getenv("ENABLE_PPROF") == "true" {
    go func() {
        log.Println(http.ListenAndServe(":6060", nil))
    }()
}
```

---

### 2. 性能影响

**CPU Profiling**：
- 开销：约 5% CPU
- 建议：不要持续采集，只在发现问题时按需采集

**Memory Profiling**：
- 开销：几乎为 0（采样机制）
- 建议：可以常驻开启

**Goroutine Profiling**：
- 开销：几乎为 0
- 建议：可以常驻开启

**Block/Mutex Profiling**：
- 开销：可能较高（取决于采样率）
- 建议：排查问题时临时启用

---

### 3. 监控告警

**方案 1：定期采集关键指标**

```bash
# cron任务，每小时采集一次goroutine数量
0 * * * * curl http://localhost:6060/debug/pprof/goroutine?debug=1 | head -1 >> /var/log/goroutine.log
```

**方案 2：集成Prometheus**

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    goroutineCount = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "go_goroutines",
        Help: "Number of goroutines",
    })
)

func init() {
    prometheus.MustRegister(goroutineCount)
}

// 定期更新指标
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        goroutineCount.Set(float64(runtime.NumGoroutine()))
    }
}()
```

**告警规则**：

```yaml
# Prometheus告警规则
groups:
  - name: golang_alerts
    rules:
      - alert: GoroutineLeaking
        expr: go_goroutines > 10000
        for: 5m
        annotations:
          summary: "Goroutine泄漏告警"
```

---

## 常见问题排查

### Q1: pprof 端口无法访问

**问题**：
```bash
$ curl http://localhost:6060/debug/pprof
curl: (7) Failed to connect to localhost port 6060: Connection refused
```

**排查步骤**：

1. 检查服务是否启动
```bash
ps aux | grep bookstore
```

2. 检查端口是否监听
```bash
lsof -i :6060
netstat -an | grep 6060
```

3. 检查防火墙
```bash
# Linux
sudo iptables -L -n

# macOS
sudo pfctl -s rules
```

---

### Q2: pprof 交互模式无法生成图形

**问题**：
```bash
(pprof) web
failed to execute dot. Is Graphviz installed?
```

**解决方法**：

```bash
# macOS
brew install graphviz

# Ubuntu
sudo apt install graphviz

# CentOS
sudo yum install graphviz
```

---

### Q3: 采集的 profile 数据为空

**问题**：
```bash
(pprof) top10
Showing nodes accounting for 0, 0% of 0 total
```

**原因**：
- 采集时间太短（默认30秒）
- 服务没有流量（CPU profile需要有负载）

**解决方法**：

```bash
# 延长采集时间
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=60

# 采集期间进行压测
wrk -t10 -c100 -d60s http://localhost:8080/api/v1/books
```

---

### Q4: 如何在测试中使用 pprof

**方法 1：测试时生成 CPU profile**

```bash
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof
```

**方法 2：测试时生成内存 profile**

```bash
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

**方法 3：集成测试中启用 pprof**

```go
func TestMain(m *testing.M) {
    // 启动pprof服务器
    go func() {
        log.Println(http.ListenAndServe(":6060", nil))
    }()
    
    // 运行测试
    code := m.Run()
    os.Exit(code)
}
```

---

## 总结

### pprof 使用清单

- [ ] **开发阶段**：集成 pprof 到 main.go
- [ ] **压测阶段**：使用 CPU profiling 定位性能瓶颈
- [ ] **上线前**：检查 goroutine 数量是否正常
- [ ] **生产环境**：限制 pprof 端口访问权限
- [ ] **问题排查**：结合火焰图和调用图分析

### 关键命令速查

```bash
# CPU分析
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 内存分析
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine分析
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Web界面（推荐）
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=30
```

### 下一步学习

- Day 21：使用 wrk 进行压力测试并优化
- 学习 Prometheus + Grafana 监控体系
- 学习分布式链路追踪（OpenTelemetry）

---

**教学要点回顾**：

1. **pprof 是性能优化的必备工具**，不要盲目猜测瓶颈
2. **CPU profiling** 找热点函数，**Memory profiling** 找内存泄漏
3. **Goroutine profiling** 检测协程泄漏，**Block/Mutex profiling** 找锁竞争
4. **火焰图** 是最直观的可视化方式
5. **生产环境** 必须限制 pprof 访问权限

记住：**过早优化是万恶之源，但没有分析的优化是盲目的！**
