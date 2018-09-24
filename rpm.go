package main

import (
	"context"
	"math"
	"periph.io/x/periph/conn/gpio"
	"time"
)

type GPIO interface {
	register(string) error
	edge(context.Context) <-chan gpio.Level
}

type rpm struct {
	context.Context
	gpio GPIO

	io   chan interface{}
	kill chan interface{}

	bufferSession time.Duration
	refreshClock  time.Duration
	correct       float64
}

func (rpm *rpm) configure(pin string) (<-chan interface{}, error) {
	if err := rpm.gpio.register(pin); err != nil {
		return nil, err
	}
	rpm.io = make(chan interface{})
	return rpm.io, nil
}

func (rpm *rpm) start() {
	rpm.kill = make(chan interface{})

	go func(kill <-chan interface{}) {
		var buffer []time.Time
		var lastLevel gpio.Level = gpio.Low
		var lastRefresh time.Time = time.Now()

		ctx, cancel := context.WithCancel(context.Background())
		edge := rpm.gpio.edge(ctx)
		for {
			select {
			case <-kill:
				cancel()
				return

			case <-time.Tick(rpm.refreshClock):
				buffer = rpm.calc(buffer)
				lastRefresh = time.Now()

			case level := <-edge:
				log.Debugf("signal received: %v", level)

				if level != lastLevel {
					lastLevel = level
					if level == gpio.Low {
						buffer = append(buffer, time.Now())
					}
				}

				if time.Since(lastRefresh) > rpm.refreshClock {
					buffer = rpm.calc(buffer)
					lastRefresh = time.Now()
				}

			}
		}
	}(rpm.kill)

	log.Debug("start session")
}

func (rpm *rpm) calc(buffer []time.Time) []time.Time {
	// remove old value
	if len(buffer) == 0 || time.Since(buffer[len(buffer)-1]) > rpm.bufferSession {
		buffer = nil
	} else {
		var i int
		for i = 0; i < len(buffer); i++ {
			if time.Since(buffer[i]) <= rpm.bufferSession {
				break
			}
		}
		buffer = buffer[i:]
	}

	var speed float64
	if len(buffer) < 2 {
		speed = 0
	} else {
		speed = float64(len(buffer)) / float64(buffer[len(buffer)-1].Sub(buffer[0])) * float64(time.Minute)
	}

	if math.IsInf(speed, 0) {
		speed = 0
	}

	select {
	case <-rpm.kill:
	case rpm.io <- speed:
		log.Debugf("RPM speed: %.2f", speed)
	}

	return buffer
}

func (rpm *rpm) stop() {
	safelyClose(&rpm.kill)
	log.Debug("stop session")
}

func (rpm *rpm) close() {
	safelyClose(&rpm.kill)
	safelyClose(&rpm.io)
	log.Debug("exit plugin")
}

func safelyClose(c *chan interface{}) {
	if *c != nil {
		close(*c)
	}
	*c = nil
}
