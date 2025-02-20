package netlistener

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func Test30SecondsRead(t *testing.T) {
	t.Run("TestGlobalBandwithRead30Seconds", func(t *testing.T) {
		globalBandwidthLimit := 100 * 1024   // 100 KB/s
		dataSentPerConnection := 1024 * 1024 // 1 MB
		expectedBandwidthConsumed := int64(globalBandwidthLimit * 30)
		allowedDeviation := 0.05

		numberOfConnections := 5
		testDeadline := time.Duration(30 * time.Second)
		totalBytesConsumed := atomic.Int64{}

		listener, err := net.Listen("tcp", ":0")
		defer listener.Close()

		listener.(*net.TCPListener).SetDeadline(time.Now().Add(500 * time.Millisecond))
		if err != nil {
			t.Fatal("Failed to create listener", err)
		}

		for i := 0; i < numberOfConnections; i++ {
			go writeDataToServer(listener, dataSentPerConnection)
		}

		throttledListener, err := NewListener(listener, &globalBandwidthLimit, nil)
		if err != nil {
			t.Fatal("Failed to create throttled listener", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), testDeadline)
		defer cancel()
		now := time.Now()

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					conn, err := throttledListener.Accept()
					if err != nil {
						continue
					}

					go func(conn net.Conn) {
						defer conn.Close()
						buf := make([]byte, 1024)
						for {
							select {
							case <-ctx.Done():
								return
							default:
								n, err := conn.Read(buf)
								if err != nil {
									if err == io.EOF {
										continue
									}
								}
								totalBytesConsumed.Add(int64(n))
							}
						}
					}(conn)
				}
			}
		}()

		wg.Wait()

		elapsed := time.Since(now)
		fmt.Printf("Total bytes consumed: %d\n", totalBytesConsumed.Load())
		fmt.Printf("Expected: %d\n", expectedBandwidthConsumed)
		fmt.Printf("ElapsedTime: %f\n", elapsed.Seconds())

		deviation := math.Abs(float64(totalBytesConsumed.Load()-expectedBandwidthConsumed)) / float64(expectedBandwidthConsumed)
		if deviation > allowedDeviation {
			t.Errorf("Deviation too high: %f", deviation)
		}
	})

	t.Run("TestGlobalBandwithRead30SecondsLimitsUpdatedInRuntime", func(t *testing.T) {
		globalBandwidthLimit := 100 * 1024        // 100 KB/s
		globalBandwidthLimitUpdated := 200 * 1024 // 200 KB/s
		dataSentPerConnection := 1024 * 1024      // 1 MB
		expectedBandwidthConsumed := int64((globalBandwidthLimit*30)/2 + (globalBandwidthLimitUpdated*30)/2)
		allowedDeviation := 0.05

		numberOfConnections := 5
		testDeadline := time.Duration(30 * time.Second)
		totalBytesConsumed := atomic.Int64{}

		listener, err := net.Listen("tcp", ":0")
		defer listener.Close()

		listener.(*net.TCPListener).SetDeadline(time.Now().Add(500 * time.Millisecond))
		if err != nil {
			t.Fatal("Failed to create listener", err)
		}

		for i := 0; i < numberOfConnections; i++ {
			go writeDataToServer(listener, dataSentPerConnection)
		}

		max := math.MaxInt
		throttledListener, err := NewListener(listener, &globalBandwidthLimit, &max)
		if err != nil {
			t.Fatal("Failed to create throttled listener", err)
		}

		// updating limits after half time has passed
		go func() {
			time.Sleep(testDeadline / 2)
			throttledListener.SetLimits(globalBandwidthLimitUpdated, math.MaxInt)
		}()

		ctx, cancel := context.WithTimeout(context.Background(), testDeadline)
		defer cancel()
		now := time.Now()

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					conn, err := throttledListener.Accept()
					if err != nil {
						continue
					}

					go func(conn net.Conn) {
						defer conn.Close()
						buf := make([]byte, 1024)
						for {
							select {
							case <-ctx.Done():
								return
							default:
								n, err := conn.Read(buf)
								if err != nil {
									if err == io.EOF {
										continue
									}
								}
								totalBytesConsumed.Add(int64(n))
							}
						}
					}(conn)
				}
			}
		}()

		wg.Wait()

		elapsed := time.Since(now)
		fmt.Printf("Total bytes consumed: %d\n", totalBytesConsumed.Load())
		fmt.Printf("Expected: %d\n", expectedBandwidthConsumed)
		fmt.Printf("ElapsedTime: %f\n", elapsed.Seconds())

		deviation := math.Abs(float64(totalBytesConsumed.Load()-expectedBandwidthConsumed)) / float64(expectedBandwidthConsumed)
		if deviation > allowedDeviation {
			t.Errorf("Deviation too high: %f", deviation)
		}
	})
}

func writeDataToServer(listener net.Listener, size int) {
	conn, _ := net.Dial("tcp", listener.Addr().String())
	defer conn.Close()

	buf := make([]byte, size)
	rand.Read(buf)
	conn.Write(buf)
}
