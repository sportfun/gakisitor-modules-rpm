package main

import (
	"context"
	"fmt"
	"github.com/thoas/go-funk"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"periph.io/x/periph/conn/gpio"
	"testing"
	"time"
)

type emulatorGPIO struct {
	sleepTime *time.Duration
}

func (*emulatorGPIO) register(string) error { return nil }
func (e *emulatorGPIO) edge(ctx context.Context) <-chan gpio.Level {
	out := make(chan gpio.Level)
	go func(out chan<- gpio.Level) {
		for {
			out <- gpio.Low
			out <- gpio.High
			out <- gpio.Low

			select {
			case <-ctx.Done():
				return
			case <-time.After(*e.sleepTime):
			}
		}
	}(out)

	return out
}

func TestRPM_Precision(t *testing.T) {
	var gAcc []float64
	var lAcc []float64
	var sleepTime = time.Duration(250 * time.Millisecond)
	var ioTick = 50 * time.Millisecond

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	ctn := true
	go func() { <-interrupt; ctn = false }()

	ctx, cancel := context.WithCancel(context.Background())
	rand.Seed(time.Now().UnixNano())
	engine := rpm{
		Context:       ctx,
		gpio:          &emulatorGPIO{sleepTime: &sleepTime},
		ioTick:        ioTick,
		io:            make(chan interface{}),
	}
	engine.start()

	for i := 0; i < 0xF && ctn; i++ {
		lAcc = nil
		rpm := rand.Int63n(125) + 75
		duration := time.Duration(rand.Intn(1500)+500) * time.Millisecond
		sleepTime = time.Duration(60 * int64(time.Second) / rpm)

		dms := float64(duration/time.Microsecond) / 1000
		sms := float64(sleepTime/time.Microsecond) / 1000
		fmt.Printf(">>> Try to target %d rpm during %.2fms (1 ping each %.2fms)\n", rpm, dms, sms)

		start := time.Now()
		for ctn {
			v := <-engine.io
			acc := math.Max(1-math.Abs(float64(rpm)-v.(float64))/float64(rpm), 0) * 100
			gAcc = append(gAcc, acc)
			lAcc = append(lAcc, acc)
			fmt.Printf("%.2f rpm\ttarget: %d rpm\taccurancy: %.3f%%\telapsed: %.2fs\n", v, rpm, acc, float64(time.Since(start)/time.Millisecond)/1000)
			if time.Since(start) > duration {
				break
			}
		}

		var latency time.Duration
		for idx, acc := range lAcc {
			if acc > 95 {
				break
			}

			latency = ioTick * time.Duration(idx+1)
		}
		localAcc := funk.SumFloat64(lAcc) / float64(len(lAcc))
		fmt.Printf("\n>>> LOCAL: Average accuracy: %.3f%%\tLatency: %v\n\n", localAcc, latency)
	}

	globalAcc := funk.SumFloat64(gAcc) / float64(len(gAcc))
	fmt.Printf("\n\n>>> TOTAL: Average accuracy: %.3f%%\n\n", globalAcc)
	cancel()

	if globalAcc < 90 {
		t.Fail()
	}
}
