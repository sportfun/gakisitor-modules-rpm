package main

import (
	"context"
	"periph.io/x/periph/conn/gpio"
	"sync/atomic"
	"time"
)

type GPIO interface {
	register(string) error
	edge(context.Context) <-chan gpio.Level
}

type rpm struct {
	context.Context
	gpio GPIO

	io     chan interface{}
	rtnCnl func()
	value  uint64

	ioTick        time.Duration
	correct       float64
}

type rtnCtxValue []interface{}
type rtnCtxKey uint

const (
	ioTick int = iota
	ioChan
	rpmValuePtr
	gpioPtr

	rtnValue    rtnCtxKey = 0xFF
	minTimeTick           = 120 * time.Millisecond  // ~ 500rpm
	maxTimeTick           = 1200 * time.Millisecond // ~ 50 rpm
)

func (rpm *rpm) configure(pin string) (<-chan interface{}, error) {
	if err := rpm.gpio.register(pin); err != nil {
		return nil, err
	}
	rpm.io = make(chan interface{})
	return rpm.io, nil
}

func (rpm *rpm) start() {
	var rtnCtx context.Context
	values := rtnCtxValue{rpm.ioTick, rpm.io, &rpm.value, rpm.gpio}
	valuedCtx := context.WithValue(rpm.Context, rtnValue, values)

	rtnCtx, rpm.rtnCnl = context.WithCancel(valuedCtx)
	rpm.value = 0

	go rpmRoutine(rtnCtx)
	go ioRoutine(rtnCtx)
}

func (rpm *rpm) stop() {
	//rpm.rtnCnl()
	//rpm.rtnCnl = nil
}

func (rpm *rpm) close() {
	safelyClose(&rpm.io)
	log.Debug("exit plugin")
}

func rpmRoutine(rtnCtx context.Context) {
	rtnValue := rtnCtx.Value(rtnValue).(rtnCtxValue)
	valuePtr := rtnValue[rpmValuePtr].(*uint64)
	gpioEvent := rtnValue[gpioPtr].(GPIO).edge(rtnCtx)

	lastEvent := time.Now()
	afterMaxTimeTick := time.After(maxTimeTick)

	for {
		select {
		case <-rtnCtx.Done():
			return
		case <-afterMaxTimeTick:
			atomic.SwapUint64(valuePtr, 5e7)
			afterMaxTimeTick = time.After(maxTimeTick)
		case <-gpioEvent:
			since := time.Since(lastEvent)
			lastEvent = time.Now()

			// Remove some noise
			if since < minTimeTick || since > maxTimeTick {
				continue
			}

			rpm := 6e7 / since.Seconds()
			atomic.SwapUint64(valuePtr, uint64(rpm))
			afterMaxTimeTick = time.After(maxTimeTick)
		}
	}
}

func ioRoutine(rtnCtx context.Context) {
	rtnValue := rtnCtx.Value(rtnValue).(rtnCtxValue)
	ticks := time.Tick(rtnValue[ioTick].(time.Duration))
	io := rtnValue[ioChan].(chan interface{})
	valuePtr := rtnValue[rpmValuePtr].(*uint64)

	for {
		select {
		case <-rtnCtx.Done():
			return
		case <-ticks:
			io <- float64(atomic.LoadUint64(valuePtr)) / 1e6
		}
	}
}

func safelyClose(c *chan interface{}) {
	if *c != nil {
		close(*c)
	}
	*c = nil
}
