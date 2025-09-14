package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// 레이턴시 측정 구조체
type LatencyMetrics struct {
	TotalLatency      time.Duration
	NetworkLatency    time.Duration
	AppProcessLatency time.Duration
	SerializeLatency  time.Duration
	DeserializeLatency time.Duration
}

// 타임스탬프가 포함된 메시지 구조체
type TimestampedMessage struct {
	JSONRPCNotification
	ServerSentTime int64 `json:"server_sent_time"`
	MessageIndex   int   `json:"message_index"`
}

// JSON-RPC 2.0 배치 알림 구조체
type JSONRPCBatchNotification []JSONRPCNotification

// 배치된 타임스탬프 메시지 구조체
type BatchedTimestampedMessage struct {
	Batch          JSONRPCBatchNotification `json:"batch"`
	ServerSentTime int64                    `json:"server_sent_time"`
	BatchSize      int                      `json:"batch_size"`
}

// SSE용 미리 준비된 배치 메시지 (WebSocket PreparedMessage와 유사)
type SSEPreparedBatch struct {
	Template       BatchedTimestampedMessage
	SerializedData []byte // 미리 직렬화된 데이터 (타임스탬프 제외)
	BatchSize      int
}

// 메모리 풀
var (
	batchPool = sync.Pool{
		New: func() interface{} {
			return &BatchedTimestampedMessage{}
		},
	}

	bytesPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 1024)
		},
	}

	sseMessagePool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 2048) // SSE 포맷용 버퍼
		},
	}
)

type JSONRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type MessageParams struct {
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// SSE 배치 캐시 (메시지 크기별로 미리 준비된 배치)
var preparedBatches = map[int]*SSEPreparedBatch{}
var preparedBatchesMutex sync.RWMutex

// 미리 준비된 배치 생성
func createPreparedBatch(batchSize int) *SSEPreparedBatch {
	preparedBatchesMutex.Lock()
	defer preparedBatchesMutex.Unlock()

	if batch, exists := preparedBatches[batchSize]; exists {
		return batch
	}

	// 템플릿 배치 생성
	var templateBatch JSONRPCBatchNotification
	for i := 0; i < batchSize; i++ {
		notification := JSONRPCNotification{
			JSONRPC: "2.0",
			Method:  "message",
			Params: MessageParams{
				Text:      fmt.Sprintf("Template message %d", i),
				Timestamp: time.Time{}, // 실제 사용시 교체될 값
			},
		}
		templateBatch = append(templateBatch, notification)
	}

	template := BatchedTimestampedMessage{
		Batch:          templateBatch,
		ServerSentTime: 0, // 실제 사용시 교체될 값
		BatchSize:      batchSize,
	}

	// 미리 직렬화 (타임스탬프는 나중에 교체)
	serializedData, err := json.Marshal(template)
	if err != nil {
		panic(err)
	}

	prepared := &SSEPreparedBatch{
		Template:       template,
		SerializedData: serializedData,
		BatchSize:      batchSize,
	}

	preparedBatches[batchSize] = prepared
	return prepared
}

// WebSocket 서버 핸들러 (레이턴시 측정용)
func websocketLatencyHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	messageCount := 1000
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		fmt.Sscanf(countParam, "%d", &messageCount)
	}

	for i := 0; i < messageCount; i++ {
		// 단일 시간 소스 사용
		sendTime := time.Now()

		message := TimestampedMessage{
			JSONRPCNotification: JSONRPCNotification{
				JSONRPC: "2.0",
				Method:  "message",
				Params: MessageParams{
					Text:      "Hello World from WebSocket",
					Timestamp: sendTime,
				},
			},
			ServerSentTime: sendTime.UnixNano(),
			MessageIndex:   i,
		}

		// 직렬화 시간 측정
		serializeStart := time.Now()
		data, err := json.Marshal(message)
		if err != nil {
			break
		}
		serializeTime := time.Since(serializeStart)

		// 네트워크 전송 시간 측정 (통일된 방식)
		networkStart := time.Now()
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			break
		}
		networkTime := time.Since(networkStart)

		// 서버 처리 총 시간
		totalServerTime := time.Since(sendTime)

		// 처리 시간 정보를 로그로 기록 (실제 환경에서는 메트릭 수집)
		_ = serializeTime
		_ = networkTime
		_ = totalServerTime
	}
}

// 기존 WebSocket 서버 핸들러
func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	messageCount := 1000
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		fmt.Sscanf(countParam, "%d", &messageCount)
	}

	for i := 0; i < messageCount; i++ {
		notification := JSONRPCNotification{
			JSONRPC: "2.0",
			Method:  "message",
			Params: MessageParams{
				Text:      "Hello World from WebSocket",
				Timestamp: time.Now(),
			},
		}

		if err := conn.WriteJSON(notification); err != nil {
			break
		}
	}
}

// SSE 서버 핸들러 (레이턴시 측정용)
func sseLatencyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	messageCount := 1000
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		fmt.Sscanf(countParam, "%d", &messageCount)
	}

	for i := 0; i < messageCount; i++ {
		// 단일 시간 소스 사용
		sendTime := time.Now()

		message := TimestampedMessage{
			JSONRPCNotification: JSONRPCNotification{
				JSONRPC: "2.0",
				Method:  "message",
				Params: MessageParams{
					Text:      "Hello World from SSE",
					Timestamp: sendTime,
				},
			},
			ServerSentTime: sendTime.UnixNano(),
			MessageIndex:   i,
		}

		// 직렬화 시간 측정 (WebSocket과 통일)
		serializeStart := time.Now()
		data, err := json.Marshal(message)
		if err != nil {
			break
		}
		serializeTime := time.Since(serializeStart)

		// 네트워크 전송 시간 측정 (통일된 방식)
		networkStart := time.Now()
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		networkTime := time.Since(networkStart)

		// 서버 처리 총 시간
		totalServerTime := time.Since(sendTime)

		// 처리 시간 정보를 로그로 기록 (실제 환경에서는 메트릭 수집)
		_ = serializeTime
		_ = networkTime
		_ = totalServerTime
	}
}

// 기존 SSE 서버 핸들러
func sseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	messageCount := 1000
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		fmt.Sscanf(countParam, "%d", &messageCount)
	}

	for i := 0; i < messageCount; i++ {
		notification := JSONRPCNotification{
			JSONRPC: "2.0",
			Method:  "message",
			Params: MessageParams{
				Text:      "Hello World from SSE",
				Timestamp: time.Now(),
			},
		}

		data, err := json.Marshal(notification)
		if err != nil {
			break
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// 최적화된 SSE 핸들러 (HTTP/2 + 배치 처리)
func optimizedSSEHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	messageCount := 1000
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		fmt.Sscanf(countParam, "%d", &messageCount)
	}

	batchSize := 10 // 10개씩 배치 처리
	if batchParam := r.URL.Query().Get("batch"); batchParam != "" {
		fmt.Sscanf(batchParam, "%d", &batchSize)
	}

	for i := 0; i < messageCount; i += batchSize {
		// 단일 시간 소스 사용
		sendTime := time.Now()

		// 배치 생성
		var batch JSONRPCBatchNotification
		actualBatchSize := batchSize
		if i+batchSize > messageCount {
			actualBatchSize = messageCount - i
		}

		for j := 0; j < actualBatchSize; j++ {
			notification := JSONRPCNotification{
				JSONRPC: "2.0",
				Method:  "message",
				Params: MessageParams{
					Text:      fmt.Sprintf("Batch message %d-%d", i, j),
					Timestamp: sendTime,
				},
			}
			batch = append(batch, notification)
		}

		batchMessage := BatchedTimestampedMessage{
			Batch:          batch,
			ServerSentTime: sendTime.UnixNano(),
			BatchSize:      actualBatchSize,
		}

		// 직렬화 및 전송
		data, err := json.Marshal(batchMessage)
		if err != nil {
			break
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// 메모리 최적화된 SSE 핸들러 (PreparedBatch 사용)
func memoryOptimizedSSEHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	messageCount := 1000
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		fmt.Sscanf(countParam, "%d", &messageCount)
	}

	batchSize := 10
	if batchParam := r.URL.Query().Get("batch"); batchParam != "" {
		fmt.Sscanf(batchParam, "%d", &batchSize)
	}

	// 미리 준비된 배치 가져오기
	preparedBatch := createPreparedBatch(batchSize)

	for i := 0; i < messageCount; i += batchSize {
		sendTime := time.Now()

		// 메모리 풀에서 버퍼 가져오기
		sseBuffer := sseMessagePool.Get().([]byte)
		sseBuffer = sseBuffer[:0] // 길이 리셋

		// 미리 직렬화된 데이터 복사 및 타임스탬프 교체
		serializedData := make([]byte, len(preparedBatch.SerializedData))
		copy(serializedData, preparedBatch.SerializedData)

		// 타임스탬프 교체 (간단한 문자열 치환)
		timestampStr := fmt.Sprintf(`"server_sent_time":%d`, sendTime.UnixNano())
		serializedData = []byte(strings.Replace(string(serializedData), `"server_sent_time":0`, timestampStr, 1))

		// 현재 시간으로 Timestamp 필드들 교체
		timeStr := sendTime.Format(time.RFC3339Nano)
		serializedData = []byte(strings.Replace(string(serializedData), `"timestamp":"0001-01-01T00:00:00Z"`, fmt.Sprintf(`"timestamp":"%s"`, timeStr), -1))

		// SSE 포맷으로 래핑
		sseBuffer = append(sseBuffer, "data: "...)
		sseBuffer = append(sseBuffer, serializedData...)
		sseBuffer = append(sseBuffer, "\n\n"...)

		// 전송
		w.Write(sseBuffer)
		flusher.Flush()

		// 버퍼 반환
		sseMessagePool.Put(sseBuffer)
	}
}

// 테스트용 HTTP 서버 시작
func startTestServer() (string, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", websocketHandler)
	mux.HandleFunc("/ws-latency", websocketLatencyHandler)
	mux.HandleFunc("/sse", sseHandler)
	mux.HandleFunc("/sse-latency", sseLatencyHandler)
	mux.HandleFunc("/sse-optimized", optimizedSSEHandler)
	mux.HandleFunc("/sse-memory-optimized", memoryOptimizedSSEHandler)

	server := &http.Server{Handler: mux}

	go func() {
		server.Serve(listener)
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", listener.Addr().(*net.TCPAddr).Port)

	return baseURL, func() {
		server.Shutdown(context.Background())
	}
}

// HTTP/2 Clear Text (h2c) 지원 테스트 서버 시작
func startH2CTestServer() (string, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", websocketHandler)
	mux.HandleFunc("/ws-latency", websocketLatencyHandler)
	mux.HandleFunc("/sse", sseHandler)
	mux.HandleFunc("/sse-latency", sseLatencyHandler)
	mux.HandleFunc("/sse-optimized", optimizedSSEHandler)
	mux.HandleFunc("/sse-memory-optimized", memoryOptimizedSSEHandler)

	// h2c 핸들러로 HTTP/2 Clear Text 지원
	h2cHandler := h2c.NewHandler(mux, &http2.Server{})

	server := &http.Server{
		Handler: h2cHandler,
	}

	go func() {
		server.Serve(listener)
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", listener.Addr().(*net.TCPAddr).Port)

	return baseURL, func() {
		server.Shutdown(context.Background())
	}
}

// WebSocket 클라이언트
func benchmarkWebSocketClient(b *testing.B, serverURL string, messageCount int) (time.Duration, int, error) {
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + fmt.Sprintf("/ws?count=%d", messageCount)

	start := time.Now()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return 0, 0, err
	}
	defer conn.Close()

	receivedCount := 0
	totalBytes := 0

	for {
		var notification JSONRPCNotification
		err := conn.ReadJSON(&notification)
		if err != nil {
			break
		}
		receivedCount++

		// 메시지 크기 계산 (대략적)
		data, _ := json.Marshal(notification)
		totalBytes += len(data)

		if receivedCount >= messageCount {
			break
		}
	}

	duration := time.Since(start)
	return duration, totalBytes, nil
}

// SSE 클라이언트
func benchmarkSSEClient(b *testing.B, serverURL string, messageCount int) (time.Duration, int, error) {
	sseURL := serverURL + fmt.Sprintf("/sse?count=%d", messageCount)

	start := time.Now()
	resp, err := http.Get(sseURL)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	receivedCount := 0
	totalBytes := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			totalBytes += len(data)
			receivedCount++

			if receivedCount >= messageCount {
				break
			}
		}
	}

	duration := time.Since(start)
	return duration, totalBytes, nil
}

// 최적화된 SSE 클라이언트 (배치 처리)
func benchmarkOptimizedSSEClient(b *testing.B, serverURL string, messageCount int, batchSize int) (time.Duration, int, error) {
	sseURL := serverURL + fmt.Sprintf("/sse-optimized?count=%d&batch=%d", messageCount, batchSize)

	// HTTP/1.1 클라이언트 (기본)
	start := time.Now()
	resp, err := http.Get(sseURL)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	receivedMessages := 0
	totalBytes := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			totalBytes += len(data)

			// 배치 메시지 파싱
			var batchMessage BatchedTimestampedMessage
			if err := json.Unmarshal([]byte(data), &batchMessage); err != nil {
				continue
			}

			receivedMessages += batchMessage.BatchSize

			if receivedMessages >= messageCount {
				break
			}
		}
	}

	duration := time.Since(start)
	return duration, totalBytes, nil
}

// HTTP/2 최적화된 SSE 클라이언트 (배치 처리)
func benchmarkH2COptimizedSSEClient(b *testing.B, serverURL string, messageCount int, batchSize int) (time.Duration, int, error) {
	sseURL := serverURL + fmt.Sprintf("/sse-optimized?count=%d&batch=%d", messageCount, batchSize)

	// HTTP/2 클라이언트 설정 (h2c)
	client := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true, // h2c를 위한 HTTP 허용
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				// TLS 없이 TCP 연결 사용
				return net.Dial(network, addr)
			},
		},
	}

	start := time.Now()
	req, err := http.NewRequest("GET", sseURL, nil)
	if err != nil {
		return 0, 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	receivedMessages := 0
	totalBytes := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			totalBytes += len(data)

			// 배치 메시지 파싱
			var batchMessage BatchedTimestampedMessage
			if err := json.Unmarshal([]byte(data), &batchMessage); err != nil {
				continue
			}

			receivedMessages += batchMessage.BatchSize

			if receivedMessages >= messageCount {
				break
			}
		}
	}

	duration := time.Since(start)
	return duration, totalBytes, nil
}

// 메모리 최적화된 SSE 클라이언트
func benchmarkMemoryOptimizedSSEClient(b *testing.B, serverURL string, messageCount int, batchSize int) (time.Duration, int, error) {
	sseURL := serverURL + fmt.Sprintf("/sse-memory-optimized?count=%d&batch=%d", messageCount, batchSize)

	start := time.Now()
	resp, err := http.Get(sseURL)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	receivedMessages := 0
	totalBytes := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			totalBytes += len(data)

			// 배치 메시지 파싱
			var batchMessage BatchedTimestampedMessage
			if err := json.Unmarshal([]byte(data), &batchMessage); err != nil {
				continue
			}

			receivedMessages += batchMessage.BatchSize

			if receivedMessages >= messageCount {
				break
			}
		}
	}

	duration := time.Since(start)
	return duration, totalBytes, nil
}

// WebSocket 벤치마크 테스트
func BenchmarkWebSocket_100Messages(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond) // 서버 시작 대기

	// 워밍업 라운드 (JIT 컴파일 및 초기화 효과 제거)
	for i := 0; i < 3; i++ {
		benchmarkWebSocketClient(b, serverURL, 10)
	}

	b.ResetTimer()

	var totalDuration time.Duration
	var totalBytes int

	for i := 0; i < b.N; i++ {
		duration, bytes, err := benchmarkWebSocketClient(b, serverURL, 100)
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

func BenchmarkWebSocket_1000Messages(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	// 워밍업 라운드
	for i := 0; i < 3; i++ {
		benchmarkWebSocketClient(b, serverURL, 10)
	}

	b.ResetTimer()

	var totalDuration time.Duration
	var totalBytes int

	for i := 0; i < b.N; i++ {
		duration, bytes, err := benchmarkWebSocketClient(b, serverURL, 1000)
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

// SSE 벤치마크 테스트
func BenchmarkSSE_100Messages(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	// 워밍업 라운드
	for i := 0; i < 3; i++ {
		benchmarkSSEClient(b, serverURL, 10)
	}

	b.ResetTimer()

	var totalDuration time.Duration
	var totalBytes int

	for i := 0; i < b.N; i++ {
		duration, bytes, err := benchmarkSSEClient(b, serverURL, 100)
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

func BenchmarkSSE_1000Messages(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	// 워밍업 라운드
	for i := 0; i < 3; i++ {
		benchmarkSSEClient(b, serverURL, 10)
	}

	b.ResetTimer()

	var totalDuration time.Duration
	var totalBytes int

	for i := 0; i < b.N; i++ {
		duration, bytes, err := benchmarkSSEClient(b, serverURL, 1000)
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

// 동시성 테스트
func BenchmarkWebSocket_Concurrent(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()

	concurrency := 10
	messagesPerClient := 100

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		start := time.Now()

		for j := 0; j < concurrency; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				benchmarkWebSocketClient(b, serverURL, messagesPerClient)
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		b.ReportMetric(float64(duration.Nanoseconds())/1e6, "ms/op")
		b.ReportMetric(float64(concurrency*messagesPerClient), "total_messages")
	}
}

func BenchmarkSSE_Concurrent(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()

	concurrency := 10
	messagesPerClient := 100

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		start := time.Now()

		for j := 0; j < concurrency; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				benchmarkSSEClient(b, serverURL, messagesPerClient)
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		b.ReportMetric(float64(duration.Nanoseconds())/1e6, "ms/op")
		b.ReportMetric(float64(concurrency*messagesPerClient), "total_messages")
	}
}

// 10,000 메시지 벤치마크 테스트
func BenchmarkWebSocket_10000Messages(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()

	var totalDuration time.Duration
	var totalBytes int

	for i := 0; i < b.N; i++ {
		duration, bytes, err := benchmarkWebSocketClient(b, serverURL, 10000)
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

func BenchmarkSSE_10000Messages(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()

	var totalDuration time.Duration
	var totalBytes int

	for i := 0; i < b.N; i++ {
		duration, bytes, err := benchmarkSSEClient(b, serverURL, 10000)
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

// 레이턴시 상세 분석용 WebSocket 클라이언트
func benchmarkWebSocketLatencyClient(b *testing.B, serverURL string, messageCount int) (LatencyMetrics, error) {
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + fmt.Sprintf("/ws-latency?count=%d", messageCount)

	// 연결 설정 시간 측정 (분리된 메트릭)
	connectStart := time.Now()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return LatencyMetrics{}, err
	}
	defer conn.Close()
	connectTime := time.Since(connectStart)

	var totalLatency, networkLatency, appProcessLatency, deserializeLatency time.Duration
	receivedCount := 0

	clientReceiveStart := time.Now()

	for receivedCount < messageCount {
		// 메시지 수신 시작 시간
		messageReceiveStart := time.Now()

		// 수동 역직렬화로 SSE와 통일 (ReadJSON 대신 수동 처리)
		deserializeStart := time.Now()
		_, rawData, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var message TimestampedMessage
		if err := json.Unmarshal(rawData, &message); err != nil {
			continue
		}
		deserializeTime := time.Since(deserializeStart)

		// 메시지 수신 완료 시간
		messageReceiveEnd := time.Now()
		clientReceiveTime := messageReceiveEnd.Sub(messageReceiveStart)

		// 서버에서 보낸 시간으로부터 네트워크 레이턴시 계산 (정확한 RTT)
		serverSentTime := time.Unix(0, message.ServerSentTime)
		networkTime := messageReceiveStart.Sub(serverSentTime)

		// 네트워크 레이턴시가 음수인 경우 시스템 클록 동기화 문제로 간주
		if networkTime < 0 {
			networkTime = 0
		}

		totalLatency += clientReceiveTime
		networkLatency += networkTime
		deserializeLatency += deserializeTime
		receivedCount++
	}

	totalProcessTime := time.Since(clientReceiveStart)
	appProcessLatency = totalProcessTime - networkLatency

	return LatencyMetrics{
		TotalLatency:       totalLatency / time.Duration(receivedCount),
		NetworkLatency:     networkLatency / time.Duration(receivedCount),
		AppProcessLatency:  appProcessLatency / time.Duration(receivedCount),
		SerializeLatency:   connectTime, // 연결 시간은 별도 측정 (WebSocket)
		DeserializeLatency: deserializeLatency / time.Duration(receivedCount),
	}, nil
}

// 레이턴시 상세 분석용 SSE 클라이언트
func benchmarkSSELatencyClient(b *testing.B, serverURL string, messageCount int) (LatencyMetrics, error) {
	sseURL := serverURL + fmt.Sprintf("/sse-latency?count=%d", messageCount)

	// 연결 설정 시간 측정
	connectStart := time.Now()
	resp, err := http.Get(sseURL)
	if err != nil {
		return LatencyMetrics{}, err
	}
	defer resp.Body.Close()
	connectTime := time.Since(connectStart)

	scanner := bufio.NewScanner(resp.Body)
	var totalLatency, networkLatency, appProcessLatency, deserializeLatency time.Duration
	receivedCount := 0

	clientReceiveStart := time.Now()

	for scanner.Scan() && receivedCount < messageCount {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			// 메시지 수신 시작 시간
			messageReceiveStart := time.Now()

			// 역직렬화 시간 측정
			deserializeStart := time.Now()
			data := strings.TrimPrefix(line, "data: ")
			var message TimestampedMessage
			if err := json.Unmarshal([]byte(data), &message); err != nil {
				continue
			}
			deserializeTime := time.Since(deserializeStart)

			// 메시지 수신 완료 시간
			messageReceiveEnd := time.Now()
			clientReceiveTime := messageReceiveEnd.Sub(messageReceiveStart)

			// 서버에서 보낸 시간으로부터 네트워크 레이턴시 계산 (정확한 RTT)
			serverSentTime := time.Unix(0, message.ServerSentTime)
			networkTime := messageReceiveStart.Sub(serverSentTime)

			// 네트워크 레이턴시가 음수인 경우 시스템 클록 동기화 문제로 간주
			if networkTime < 0 {
				networkTime = 0
			}

			totalLatency += clientReceiveTime
			networkLatency += networkTime
			deserializeLatency += deserializeTime
			receivedCount++
		}
	}

	totalProcessTime := time.Since(clientReceiveStart)
	appProcessLatency = totalProcessTime - networkLatency

	return LatencyMetrics{
		TotalLatency:       totalLatency / time.Duration(receivedCount),
		NetworkLatency:     networkLatency / time.Duration(receivedCount),
		AppProcessLatency:  appProcessLatency / time.Duration(receivedCount),
		SerializeLatency:   connectTime, // 연결 시간은 별도 측정 (SSE)
		DeserializeLatency: deserializeLatency / time.Duration(receivedCount),
	}, nil
}

// WebSocket 레이턴시 상세 분석 벤치마크
func BenchmarkWebSocketLatency_1000Messages(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()

	var totalMetrics LatencyMetrics

	for i := 0; i < b.N; i++ {
		metrics, err := benchmarkWebSocketLatencyClient(b, serverURL, 1000)
		if err != nil {
			b.Fatal(err)
		}

		totalMetrics.TotalLatency += metrics.TotalLatency
		totalMetrics.NetworkLatency += metrics.NetworkLatency
		totalMetrics.AppProcessLatency += metrics.AppProcessLatency
		totalMetrics.SerializeLatency += metrics.SerializeLatency
		totalMetrics.DeserializeLatency += metrics.DeserializeLatency
	}

	avgMetrics := LatencyMetrics{
		TotalLatency:       totalMetrics.TotalLatency / time.Duration(b.N),
		NetworkLatency:     totalMetrics.NetworkLatency / time.Duration(b.N),
		AppProcessLatency:  totalMetrics.AppProcessLatency / time.Duration(b.N),
		SerializeLatency:   totalMetrics.SerializeLatency / time.Duration(b.N),
		DeserializeLatency: totalMetrics.DeserializeLatency / time.Duration(b.N),
	}

	b.ReportMetric(float64(avgMetrics.TotalLatency.Nanoseconds())/1e6, "total_latency_ms")
	b.ReportMetric(float64(avgMetrics.NetworkLatency.Nanoseconds())/1e6, "network_latency_ms")
	b.ReportMetric(float64(avgMetrics.AppProcessLatency.Nanoseconds())/1e6, "app_process_latency_ms")
	b.ReportMetric(float64(avgMetrics.DeserializeLatency.Nanoseconds())/1e6, "deserialize_latency_ms")

	// 네트워크 레이턴시 점유율 계산
	networkRatio := float64(avgMetrics.NetworkLatency) / float64(avgMetrics.TotalLatency) * 100
	appRatio := float64(avgMetrics.AppProcessLatency) / float64(avgMetrics.TotalLatency) * 100

	b.ReportMetric(networkRatio, "network_latency_percent")
	b.ReportMetric(appRatio, "app_latency_percent")
}

// SSE 레이턴시 상세 분석 벤치마크
func BenchmarkSSELatency_1000Messages(b *testing.B) {
	serverURL, cleanup := startTestServer()
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()

	var totalMetrics LatencyMetrics

	for i := 0; i < b.N; i++ {
		metrics, err := benchmarkSSELatencyClient(b, serverURL, 1000)
		if err != nil {
			b.Fatal(err)
		}

		totalMetrics.TotalLatency += metrics.TotalLatency
		totalMetrics.NetworkLatency += metrics.NetworkLatency
		totalMetrics.AppProcessLatency += metrics.AppProcessLatency
		totalMetrics.SerializeLatency += metrics.SerializeLatency
		totalMetrics.DeserializeLatency += metrics.DeserializeLatency
	}

	avgMetrics := LatencyMetrics{
		TotalLatency:       totalMetrics.TotalLatency / time.Duration(b.N),
		NetworkLatency:     totalMetrics.NetworkLatency / time.Duration(b.N),
		AppProcessLatency:  totalMetrics.AppProcessLatency / time.Duration(b.N),
		SerializeLatency:   totalMetrics.SerializeLatency / time.Duration(b.N),
		DeserializeLatency: totalMetrics.DeserializeLatency / time.Duration(b.N),
	}

	b.ReportMetric(float64(avgMetrics.TotalLatency.Nanoseconds())/1e6, "total_latency_ms")
	b.ReportMetric(float64(avgMetrics.NetworkLatency.Nanoseconds())/1e6, "network_latency_ms")
	b.ReportMetric(float64(avgMetrics.AppProcessLatency.Nanoseconds())/1e6, "app_process_latency_ms")
	b.ReportMetric(float64(avgMetrics.DeserializeLatency.Nanoseconds())/1e6, "deserialize_latency_ms")

	// 네트워크 레이턴시 점유율 계산
	networkRatio := float64(avgMetrics.NetworkLatency) / float64(avgMetrics.TotalLatency) * 100
	appRatio := float64(avgMetrics.AppProcessLatency) / float64(avgMetrics.TotalLatency) * 100

	b.ReportMetric(networkRatio, "network_latency_percent")
	b.ReportMetric(appRatio, "app_latency_percent")
}