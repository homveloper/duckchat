package main

import (
	"testing"
	"time"
)

// 최적화된 SSE 벤치마크 테스트들
func BenchmarkOptimizedSSE_100Messages_Batch10(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	// 워밍업 라운드
	for i := 0; i < 3; i++ {
		benchmarkOptimizedSSEClient(b, serverURL, 10, 10)
	}

	b.ResetTimer()

	var totalDuration time.Duration
	var totalBytes int

	for i := 0; i < b.N; i++ {
		duration, bytes, err := benchmarkOptimizedSSEClient(b, serverURL, 100, 10)
		if err != nil {
			b.Fatal(err)
		}
		totalDuration += duration
		totalBytes += bytes
	}

	avgDuration := totalDuration / time.Duration(b.N)
	avgBytes := totalBytes / b.N

	b.ReportMetric(float64(avgDuration.Nanoseconds())/1e6, "ms/op")
	b.ReportMetric(float64(avgBytes), "bytes/op")
	b.ReportMetric(float64(avgBytes)/avgDuration.Seconds(), "bytes/sec")
}

func BenchmarkOptimizedSSE_1000Messages_Batch10(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	// 워밍업 라운드
	for i := 0; i < 3; i++ {
		benchmarkOptimizedSSEClient(b, serverURL, 10, 10)
	}

	b.ResetTimer()

	var totalDuration time.Duration
	var totalBytes int

	for i := 0; i < b.N; i++ {
		duration, bytes, err := benchmarkOptimizedSSEClient(b, serverURL, 1000, 10)
		if err != nil {
			b.Fatal(err)
		}
		totalDuration += duration
		totalBytes += bytes
	}

	avgDuration := totalDuration / time.Duration(b.N)
	avgBytes := totalBytes / b.N

	b.ReportMetric(float64(avgDuration.Nanoseconds())/1e6, "ms/op")
	b.ReportMetric(float64(avgBytes), "bytes/op")
	b.ReportMetric(float64(avgBytes)/avgDuration.Seconds(), "bytes/sec")
}

func BenchmarkOptimizedSSE_1000Messages_Batch50(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	// 워밍업 라운드
	for i := 0; i < 3; i++ {
		benchmarkOptimizedSSEClient(b, serverURL, 10, 50)
	}

	b.ResetTimer()

	var totalDuration time.Duration
	var totalBytes int

	for i := 0; i < b.N; i++ {
		duration, bytes, err := benchmarkOptimizedSSEClient(b, serverURL, 1000, 50)
		if err != nil {
			b.Fatal(err)
		}
		totalDuration += duration
		totalBytes += bytes
	}

	avgDuration := totalDuration / time.Duration(b.N)
	avgBytes := totalBytes / b.N

	b.ReportMetric(float64(avgDuration.Nanoseconds())/1e6, "ms/op")
	b.ReportMetric(float64(avgBytes), "bytes/op")
	b.ReportMetric(float64(avgBytes)/avgDuration.Seconds(), "bytes/sec")
}

func BenchmarkOptimizedSSE_10000Messages_Batch50(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	// 워밍업 라운드
	for i := 0; i < 3; i++ {
		benchmarkOptimizedSSEClient(b, serverURL, 100, 50)
	}

	b.ResetTimer()

	var totalDuration time.Duration
	var totalBytes int

	for i := 0; i < b.N; i++ {
		duration, bytes, err := benchmarkOptimizedSSEClient(b, serverURL, 10000, 50)
		if err != nil {
			b.Fatal(err)
		}
		totalDuration += duration
		totalBytes += bytes
	}

	avgDuration := totalDuration / time.Duration(b.N)
	avgBytes := totalBytes / b.N

	b.ReportMetric(float64(avgDuration.Nanoseconds())/1e6, "ms/op")
	b.ReportMetric(float64(avgBytes), "bytes/op")
	b.ReportMetric(float64(avgBytes)/avgDuration.Seconds(), "bytes/sec")
}