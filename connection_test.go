package netlistener

import (
	"crypto/rand"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func ptr[T any](value T) *T {
	return &value
}

func TestRateLimitedConnection_PerConnLimit_Read(t *testing.T) {
	tests := []struct {
		name           string
		perConnLimit   *int
		randomDataSize int
		bufSize        int
		assertionFunc  func(t *testing.T, elapsedTime time.Duration)
	}{
		{
			name:           "Connection is within limits, not throttled",
			perConnLimit:   ptr(20),
			randomDataSize: 15,
			bufSize:        4,
			assertionFunc: func(t *testing.T, elapsedTime time.Duration) {
				if elapsedTime.Seconds() > 1 {
					t.Errorf("expected less than 1 second, got %f", elapsedTime.Seconds())
				}
			},
		},
		{
			name:           "Connection is throttled, it should take between 2 to 3 seconds",
			perConnLimit:   ptr(20),
			randomDataSize: 50,
			bufSize:        15,
			assertionFunc: func(t *testing.T, elapsedTime time.Duration) {
				if elapsedTime.Seconds() < 2 || elapsedTime.Seconds() > 3 {
					t.Errorf("expected between 2 to 3 seconds, got %f", elapsedTime.Seconds())
				}
			},
		},
		{
			name:           "Connection has no PerConnLimit",
			perConnLimit:   nil,
			randomDataSize: 100,
			bufSize:        10,
			assertionFunc: func(t *testing.T, elapsedTime time.Duration) {
				if elapsedTime.Seconds() > 1 {
					t.Errorf("expected less than 1 second, got %f", elapsedTime.Seconds())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			connRead, connWrite := net.Pipe()
			config := NewBandwithConfig(nil, tt.perConnLimit)
			connectionConfig := NewConnectionBandwithConfig(config)
			throttledConn := NewThrottledConnection(connRead, connectionConfig)

			go writeRandomDataToConn(connWrite, tt.randomDataSize)

			start := time.Now()

			for {
				_, err := throttledConn.Read(make([]byte, tt.bufSize))
				if err != nil {
					if err == io.EOF {
						break
					}
				}
			}
			elapsedTime := time.Since(start)

			tt.assertionFunc(t, elapsedTime)
		})

	}
}

func TestRateLimitedConnection_PerConnLimit_Write(t *testing.T) {
	tests := []struct {
		name           string
		perConnLimit   *int
		numberOfChunks int
		bufSize        int
		assertionFunc  func(t *testing.T, elapsedTime time.Duration)
	}{
		{
			name:           "Connection is within limits, not throttled",
			perConnLimit:   ptr(20),
			numberOfChunks: 2,
			bufSize:        4,
			assertionFunc: func(t *testing.T, elapsedTime time.Duration) {
				if elapsedTime.Seconds() > 1 {
					t.Errorf("expected less than 1 second, got %f", elapsedTime.Seconds())
				}
			},
		},
		{
			name:           "Connection is throttled, it should take between 2 to 3 seconds",
			perConnLimit:   ptr(20),
			numberOfChunks: 3,
			bufSize:        20,
			assertionFunc: func(t *testing.T, elapsedTime time.Duration) {
				if elapsedTime.Seconds() < 2 || elapsedTime.Seconds() > 3 {
					t.Errorf("expected between 2 to 3 seconds, got %f", elapsedTime.Seconds())
				}
			},
		},
		{
			name:           "Connection has no PerConnLimit",
			perConnLimit:   nil,
			numberOfChunks: 10,
			bufSize:        10,
			assertionFunc: func(t *testing.T, elapsedTime time.Duration) {
				if elapsedTime.Seconds() > 1 {
					t.Errorf("expected less than 1 second, got %f", elapsedTime.Seconds())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			connRead, connWrite := net.Pipe()
			config := NewBandwithConfig(nil, tt.perConnLimit)
			connectionConfig := NewConnectionBandwithConfig(config)
			throttledConn := NewThrottledConnection(connWrite, connectionConfig)

			go readDataFromConn(connRead)

			start := time.Now()

			for i := 0; i < tt.numberOfChunks; i++ {
				writeRandomDataToConn(throttledConn, tt.bufSize)
			}

			elapsedTime := time.Since(start)

			tt.assertionFunc(t, elapsedTime)
		})

	}
}

func TestRateLimitedConnection_GlobalLimiter_Read(t *testing.T) {
	tests := []struct {
		name           string
		numberOfConn   int
		globalLimit    *int
		randomDataSize int
		bufSize        int
		assertionFunc  func(t *testing.T, elapsedTimeMs int64)
	}{
		{
			name:           "Single connection is within limits, not throttled",
			numberOfConn:   1,
			globalLimit:    ptr(19),
			randomDataSize: 15,
			bufSize:        4,
			assertionFunc: func(t *testing.T, elapsedTimeMs int64) {
				if elapsedTimeMs > 1000 {
					t.Errorf("expected less than 1000 ms, got %d", elapsedTimeMs)
				}
			},
		},
		{
			name:           "Two connections are within limits, not throttled",
			numberOfConn:   2,
			globalLimit:    ptr(100),
			randomDataSize: 40,
			bufSize:        20,
			assertionFunc: func(t *testing.T, elapsedTimeMs int64) {
				if elapsedTimeMs > 1000 {
					t.Errorf("expected less than 1000 ms, got %d", elapsedTimeMs)
				}
			},
		},
		{
			name:           "Two connections are throttled, it should take between 3 to 4 seconds",
			numberOfConn:   2,
			globalLimit:    ptr(100),
			randomDataSize: 90,
			bufSize:        100,
			assertionFunc: func(t *testing.T, elapsedTimeMs int64) {
				if elapsedTimeMs < 3000 || elapsedTimeMs > 4000 {
					t.Errorf("expected between 3000 to 4000 ms, got %d", elapsedTimeMs)
				}
			},
		},
		{
			name:           "Five connections are throttled, it should take between 9 to 10 seconds",
			numberOfConn:   5,
			globalLimit:    ptr(100),
			randomDataSize: 10,
			bufSize:        100,
			assertionFunc: func(t *testing.T, elapsedTimeMs int64) {
				if elapsedTimeMs < 9000 || elapsedTimeMs > 10000 {
					t.Errorf("expected between 9000 to 10000 ms, got %d", elapsedTimeMs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wg := sync.WaitGroup{}

			config := NewBandwithConfig(tt.globalLimit, nil)
			wg.Add(tt.numberOfConn)

			start := time.Now()
			maxElapsedTime := time.Duration(0)

			for i := 0; i < tt.numberOfConn; i++ {
				connRead, connWrite := net.Pipe()
				connectionConfig := NewConnectionBandwithConfig(config)
				throttledConn := NewThrottledConnection(connRead, connectionConfig)

				go writeRandomDataToConn(connWrite, tt.randomDataSize)

				go func() {
					defer wg.Done()

					for {
						_, err := throttledConn.Read(make([]byte, tt.bufSize))
						if err != nil {
							if err == io.EOF {
								break
							}
						}
					}
					maxElapsedTime = time.Since(start)
				}()
			}

			wg.Wait()

			tt.assertionFunc(t, maxElapsedTime.Milliseconds())

		})
	}
}

func TestRateLimitedConnection_GlobalLimiter_Write(t *testing.T) {
	tests := []struct {
		name           string
		numberOfConn   int
		globalLimit    *int
		randomDataSize int
		numberOfChunks int
		assertionFunc  func(t *testing.T, elapsedTimeMs int64)
	}{
		{
			name:           "Single connection is within limits, not throttled",
			numberOfConn:   1,
			globalLimit:    ptr(19),
			randomDataSize: 5,
			numberOfChunks: 3,
			assertionFunc: func(t *testing.T, elapsedTimeMs int64) {
				if elapsedTimeMs > 1000 {
					t.Errorf("expected less than 1000 ms, got %d", elapsedTimeMs)
				}
			},
		},
		{
			name:           "Two connections are within limits, not throttled",
			numberOfConn:   2,
			globalLimit:    ptr(100),
			randomDataSize: 10,
			numberOfChunks: 2,
			assertionFunc: func(t *testing.T, elapsedTimeMs int64) {
				if elapsedTimeMs > 1000 {
					t.Errorf("expected less than 1000 ms, got %d", elapsedTimeMs)
				}
			},
		},
		{
			name:           "Two connections are throttled, it should take between 3 to 4 seconds",
			numberOfConn:   2,
			globalLimit:    ptr(100),
			randomDataSize: 100,
			numberOfChunks: 2,
			assertionFunc: func(t *testing.T, elapsedTimeMs int64) {
				if elapsedTimeMs < 3000 || elapsedTimeMs > 4000 {
					t.Errorf("expected between 3000 to 4000 ms, got %d", elapsedTimeMs)
				}
			},
		},
		{
			name:           "Five connections are throttled, it should take between 4 to 5 seconds",
			numberOfConn:   5,
			globalLimit:    ptr(100),
			randomDataSize: 10,
			numberOfChunks: 10,
			assertionFunc: func(t *testing.T, elapsedTimeMs int64) {
				if elapsedTimeMs < 4000 || elapsedTimeMs > 5000 {
					t.Errorf("expected between 4000 to 5000 ms, got %d", elapsedTimeMs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wg := sync.WaitGroup{}

			config := NewBandwithConfig(tt.globalLimit, nil)
			wg.Add(tt.numberOfConn)

			start := time.Now()
			maxElapsedTime := time.Duration(0)

			for i := 0; i < tt.numberOfConn; i++ {
				connRead, connWrite := net.Pipe()
				connectionConfig := NewConnectionBandwithConfig(config)
				throttledConn := NewThrottledConnection(connWrite, connectionConfig)

				go readDataFromConn(connRead)

				go writeRandomDataToConn(connWrite, tt.randomDataSize)

				go func() {
					defer wg.Done()

					for i := 0; i < tt.numberOfChunks; i++ {
						writeRandomDataToConn(throttledConn, tt.randomDataSize)
					}

					maxElapsedTime = time.Since(start)
				}()
			}

			wg.Wait()

			tt.assertionFunc(t, maxElapsedTime.Milliseconds())

		})
	}
}

func writeRandomDataToConn(conn net.Conn, size int) {
	defer conn.Close()

	buf := make([]byte, size)
	_, _ = rand.Read(buf)
	conn.Write(buf)
}

func readDataFromConn(conn net.Conn) {
	for {
		_, err := conn.Read(make([]byte, 200))
		if err != nil {
			if err == io.EOF {
				break
			}
		}
	}
}
