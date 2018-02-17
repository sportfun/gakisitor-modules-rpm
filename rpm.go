package main

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
)

type rpm struct {
	pin        string
	clock      time.Duration
	bufferTemp time.Duration

	lastRefresh time.Time
	lastLevel   gpio.Level
	buffer      []time.Time

	mtx  sync.Mutex
	kill chan struct{}
}

var errWait = errors.New("")
var errInvalidPin = errors.New("invalid configuration pin")

func (rpm *rpm) shift() {
	rpm.mtx.Lock()
	defer rpm.mtx.Unlock()
	for len(rpm.buffer) > 0 && time.Since(rpm.buffer[0]) > rpm.bufferTemp {
		rpm.buffer = rpm.buffer[1:]
	}
}

func (rpm *rpm) get() (int64, error) {
	if time.Since(rpm.lastRefresh) < rpm.clock {
		return 0, errWait
	}

	rpm.shift()
	rpm.lastRefresh = time.Now()

	return int64(float64(len(rpm.buffer)) / float64(rpm.bufferTemp) * float64(60*time.Second)), nil
}

func (rpm *rpm) start() error {
	rpm.kill = make(chan struct{})
	p := gpioreg.ByName(rpm.pin)
	if p == nil {
		return errInvalidPin
	}

	go func(kill <-chan struct{}) {

		for {
			select {
			case <-kill:
				return
			case level := <-waitForEdge(p):
				rpm.shift()
				rpm.mtx.Lock()
				if level != rpm.lastLevel {
					rpm.lastLevel = level
					if level == gpio.Low {
						rpm.buffer = append(rpm.buffer, time.Now())
					}
				}
				rpm.mtx.Unlock()
			}
		}
	}(rpm.kill)
	return nil
}

func (rpm *rpm) stop() error {
	rpm.kill <- struct{}{}
	return nil
}

func waitForEdge(pin gpio.PinIO) <-chan gpio.Level {
	out := make(chan gpio.Level)
	go func(pin gpio.PinIO) {
		pin.WaitForEdge(-1)
		out <- pin.Read()
	}(pin)
	return out
}
