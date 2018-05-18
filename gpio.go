package main

import (
	"errors"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
)

type _gpio struct {
	context.Context
	pin gpio.PinIO
}

func (inst *_gpio) register(pin string) error {
	if _, err := host.Init(); err != nil {
		return err
	}

	inst.pin = gpioreg.ByName(pin)
	if inst.pin == nil {
		return errors.New("invalid pin: "+pin)
	}
	return inst.pin.In(gpio.PullUp, gpio.BothEdges)
//	return nil
}

func (inst *_gpio) egde() <-chan gpio.Level {
	levels := make(chan gpio.Level)
	go func(pin gpio.PinIO, levels chan<- gpio.Level) {
		for {
			pin.WaitForEdge(-1)
			logrus.Debugf("OK")
			levels <- pin.Read()
			time.Sleep(10*time.Millisecond)
		}
	}(inst.pin, levels)
	return levels
}
