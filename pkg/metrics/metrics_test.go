package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// TestInitMetrics 测试指标初始化
func TestInitMetrics(t *testing.T) {
	// 初始化指标
	InitMetrics()

	// 验证所有指标已创建
	if HTTPRequestsTotal == nil {
		t.Error("HTTPRequestsTotal未初始化")
	}
	if HTTPRequestDuration == nil {
		t.Error("HTTPRequestDuration未初始化")
	}
	if HTTPRequestsInProgress == nil {
		t.Error("HTTPRequestsInProgress未初始化")
	}

	t.Log("✅ 所有指标初始化成功")
}

// TestCounter 测试Counter指标
func TestCounter(t *testing.T) {
	InitMetrics()

	// 初始值应为0
	initialValue := getCounterValue(t, OrdersCreatedTotal)
	if initialValue != 0 {
		t.Errorf("Counter初始值错误: expected=0, got=%f", initialValue)
	}

	// 递增3次
	IncCounter(OrdersCreatedTotal)
	IncCounter(OrdersCreatedTotal)
	IncCounter(OrdersCreatedTotal)

	// 验证值为3
	value := getCounterValue(t, OrdersCreatedTotal)
	if value != 3 {
		t.Errorf("Counter值错误: expected=3, got=%f", value)
	}

	t.Log("✅ Counter测试通过")
}

// TestCounterVec 测试CounterVec指标
func TestCounterVec(t *testing.T) {
	InitMetrics()

	// 递增不同标签的Counter
	IncCounterVec(HTTPRequestsTotal, map[string]string{
		"method": "GET",
		"path":   "/api/orders",
		"status": "200",
	})

	IncCounterVec(HTTPRequestsTotal, map[string]string{
		"method": "POST",
		"path":   "/api/orders",
		"status": "201",
	})

	IncCounterVec(HTTPRequestsTotal, map[string]string{
		"method": "GET",
		"path":   "/api/orders",
		"status": "200",
	})

	// 验证GET /api/orders 200的计数为2
	labels := map[string]string{
		"method": "GET",
		"path":   "/api/orders",
		"status": "200",
	}
	value := getCounterVecValue(t, HTTPRequestsTotal, labels)
	if value != 2 {
		t.Errorf("CounterVec值错误: expected=2, got=%f", value)
	}

	t.Log("✅ CounterVec测试通过")
}

// TestGauge 测试Gauge指标
func TestGauge(t *testing.T) {
	InitMetrics()

	// 初始值应为0
	initialValue := getGaugeValue(t, HTTPRequestsInProgress)
	if initialValue != 0 {
		t.Errorf("Gauge初始值错误: expected=0, got=%f", initialValue)
	}

	// 递增
	IncGauge(HTTPRequestsInProgress)
	IncGauge(HTTPRequestsInProgress)
	value := getGaugeValue(t, HTTPRequestsInProgress)
	if value != 2 {
		t.Errorf("Gauge递增后值错误: expected=2, got=%f", value)
	}

	// 递减
	DecGauge(HTTPRequestsInProgress)
	value = getGaugeValue(t, HTTPRequestsInProgress)
	if value != 1 {
		t.Errorf("Gauge递减后值错误: expected=1, got=%f", value)
	}

	// 设置值
	SetGauge(HTTPRequestsInProgress, 10)
	value = getGaugeValue(t, HTTPRequestsInProgress)
	if value != 10 {
		t.Errorf("Gauge设置后值错误: expected=10, got=%f", value)
	}

	t.Log("✅ Gauge测试通过")
}

// TestGaugeVec 测试GaugeVec指标
func TestGaugeVec(t *testing.T) {
	InitMetrics()

	// 设置不同熔断器的状态
	SetGaugeVec(CircuitBreakerState, map[string]string{"name": "inventory-service"}, 0) // CLOSED
	SetGaugeVec(CircuitBreakerState, map[string]string{"name": "payment-service"}, 1)   // OPEN

	// 验证值
	value1 := getGaugeVecValue(t, CircuitBreakerState, map[string]string{"name": "inventory-service"})
	if value1 != 0 {
		t.Errorf("GaugeVec值错误: expected=0, got=%f", value1)
	}

	value2 := getGaugeVecValue(t, CircuitBreakerState, map[string]string{"name": "payment-service"})
	if value2 != 1 {
		t.Errorf("GaugeVec值错误: expected=1, got=%f", value2)
	}

	t.Log("✅ GaugeVec测试通过")
}

// TestHistogram 测试Histogram指标
func TestHistogram(t *testing.T) {
	InitMetrics()

	// 记录多个观测值
	ObserveHistogram(OrderCreationDuration, 0.05) // 50ms
	ObserveHistogram(OrderCreationDuration, 0.1)  // 100ms
	ObserveHistogram(OrderCreationDuration, 0.5)  // 500ms
	ObserveHistogram(OrderCreationDuration, 1.0)  // 1s
	ObserveHistogram(OrderCreationDuration, 5.0)  // 5s

	// 验证观测次数
	count := getHistogramCount(t, OrderCreationDuration)
	if count != 5 {
		t.Errorf("Histogram观测次数错误: expected=5, got=%d", count)
	}

	// 验证总和
	sum := getHistogramSum(t, OrderCreationDuration)
	expectedSum := 0.05 + 0.1 + 0.5 + 1.0 + 5.0
	if sum != expectedSum {
		t.Errorf("Histogram总和错误: expected=%f, got=%f", expectedSum, sum)
	}

	t.Logf("✅ Histogram测试通过, 观测次数=%d, 总和=%f, 平均值=%f", count, sum, sum/float64(count))
}

// TestHistogramVec 测试HistogramVec指标
func TestHistogramVec(t *testing.T) {
	InitMetrics()

	// 记录不同路径的请求耗时
	ObserveHistogramVec(HTTPRequestDuration, map[string]string{"method": "GET", "path": "/api/orders"}, 0.05)
	ObserveHistogramVec(HTTPRequestDuration, map[string]string{"method": "GET", "path": "/api/orders"}, 0.1)
	ObserveHistogramVec(HTTPRequestDuration, map[string]string{"method": "POST", "path": "/api/orders"}, 0.2)

	// 验证GET /api/orders的观测次数
	labels := map[string]string{"method": "GET", "path": "/api/orders"}
	count := getHistogramVecCount(t, HTTPRequestDuration, labels)
	if count != 2 {
		t.Errorf("HistogramVec观测次数错误: expected=2, got=%d", count)
	}

	t.Log("✅ HistogramVec测试通过")
}

// TestRealWorldScenario 真实场景：模拟HTTP请求处理
func TestRealWorldScenario(t *testing.T) {
	InitMetrics()

	// 重置Gauge（清理之前测试的影响）
	SetGauge(HTTPRequestsInProgress, 0)

	// 模拟10个HTTP请求
	for i := 0; i < 10; i++ {
		// 递增正在处理的请求数
		IncGauge(HTTPRequestsInProgress)

		// 模拟处理耗时
		start := time.Now()
		time.Sleep(10 * time.Millisecond)
		duration := time.Since(start).Seconds()

		// 记录耗时
		ObserveHistogramVec(HTTPRequestDuration, map[string]string{
			"method": "POST",
			"path":   "/api/orders",
		}, duration)

		// 递增请求总数
		IncCounterVec(HTTPRequestsTotal, map[string]string{
			"method": "POST",
			"path":   "/api/orders",
			"status": "200",
		})

		// 递减正在处理的请求数
		DecGauge(HTTPRequestsInProgress)
	}

	// 验证正在处理的请求数（应为0）
	inProgress := getGaugeValue(t, HTTPRequestsInProgress)
	if inProgress != 0 {
		t.Errorf("正在处理的请求数错误: expected=0, got=%f", inProgress)
	}

	t.Log("✅ 真实场景测试通过")
	t.Log("   提示: 启动Prometheus和Grafana后可在Dashboard中查看这些指标")
}

// 辅助函数：获取Counter值
func getCounterValue(t *testing.T, counter prometheus.Counter) float64 {
	var metric dto.Metric
	if err := counter.Write(&metric); err != nil {
		t.Fatalf("读取Counter值失败: %v", err)
	}
	return metric.Counter.GetValue()
}

// 辅助函数：获取CounterVec值
func getCounterVecValue(t *testing.T, counterVec *prometheus.CounterVec, labels map[string]string) float64 {
	var metric dto.Metric
	counter := counterVec.With(labels)
	if err := counter.(prometheus.Counter).Write(&metric); err != nil {
		t.Fatalf("读取CounterVec值失败: %v", err)
	}
	return metric.Counter.GetValue()
}

// 辅助函数：获取Gauge值
func getGaugeValue(t *testing.T, gauge prometheus.Gauge) float64 {
	var metric dto.Metric
	if err := gauge.Write(&metric); err != nil {
		t.Fatalf("读取Gauge值失败: %v", err)
	}
	return metric.Gauge.GetValue()
}

// 辅助函数：获取GaugeVec值
func getGaugeVecValue(t *testing.T, gaugeVec *prometheus.GaugeVec, labels map[string]string) float64 {
	var metric dto.Metric
	gauge := gaugeVec.With(labels)
	if err := gauge.(prometheus.Gauge).Write(&metric); err != nil {
		t.Fatalf("读取GaugeVec值失败: %v", err)
	}
	return metric.Gauge.GetValue()
}

// 辅助函数：获取Histogram观测次数
func getHistogramCount(t *testing.T, histogram prometheus.Histogram) uint64 {
	var metric dto.Metric
	if err := histogram.Write(&metric); err != nil {
		t.Fatalf("读取Histogram值失败: %v", err)
	}
	return metric.Histogram.GetSampleCount()
}

// 辅助函数：获取Histogram总和
func getHistogramSum(t *testing.T, histogram prometheus.Histogram) float64 {
	var metric dto.Metric
	if err := histogram.Write(&metric); err != nil {
		t.Fatalf("读取Histogram值失败: %v", err)
	}
	return metric.Histogram.GetSampleSum()
}

// 辅助函数：获取HistogramVec观测次数
func getHistogramVecCount(t *testing.T, histogramVec *prometheus.HistogramVec, labels map[string]string) uint64 {
	var metric dto.Metric
	histogram := histogramVec.With(labels)
	if err := histogram.(prometheus.Histogram).Write(&metric); err != nil {
		t.Fatalf("读取HistogramVec值失败: %v", err)
	}
	return metric.Histogram.GetSampleCount()
}
