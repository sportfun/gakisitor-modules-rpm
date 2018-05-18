package main

import (
	"context"
	"periph.io/x/periph/conn/gpio"
	"time"

	"github.com/sirupsen/logrus"
)

type GPIO interface {
	register(string) error
	egde() <-chan gpio.Level
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

// TODO: Priorize tick (select -> default -> select)
	go func(kill <-chan interface{}) {
		var buffer []time.Time
		var lastLevel gpio.Level = gpio.Low
		var lastRefresh time.Time = time.Now()


		edge := rpm.gpio.egde()
		for {
			select {
			case <-kill:
				return

			case <-time.Tick(rpm.refreshClock):
				buffer = rpm.calc(buffer)
				lastRefresh = time.Now()

			case level := <-edge:
				logrus.Debugf("LEVEL: %v", level)
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
		logrus.Debugf("REMOVE %d ITEM: %v => %v < %v", i, buffer[i], time.Since(buffer[i]), rpm.bufferSession)
		buffer = buffer[i:]
	}

	var speed float64
	if len(buffer) < 2 {
		speed = 0
	} else {
		speed = float64(len(buffer)) / float64(buffer[len(buffer)-1].Sub(buffer[0])) * float64(time.Minute)
	}

	if buffer != nil {
		logrus.Debugf("SPEED: %v (%v/%v)", speed, len(buffer), float64(buffer[len(buffer)-1].Sub(buffer[0])))
	}
	select {
	case <-rpm.kill:
	case rpm.io <- speed:
	}

	return buffer
}

func (rpm *rpm) stop() {
	safelyClose(&rpm.kill)
}

func (rpm *rpm) close() {
	safelyClose(&rpm.kill)
	safelyClose(&rpm.io)
}

func safelyClose(c *chan interface{}) {
	if *c != nil {
		close(*c)
	}
	*c = nil
}
