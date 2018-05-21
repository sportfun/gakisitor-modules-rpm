package main

import (
	"context"
	"errors"

	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
)

type _gpio struct {
	pin gpio.PinIO
}

func (inst *_gpio) register(pin string) error {
	if _, err := host.Init(); err != nil {
		return err
	}

	inst.pin = gpioreg.ByName(pin)
	if inst.pin == nil {
		return errors.New("invalid pin: " + pin)
	}
	return inst.pin.In(gpio.PullUp, gpio.BothEdges)
}

func (inst *_gpio) edge(ctx context.Context) <-chan gpio.Level {
	levels := make(chan gpio.Level)
	go func(pin gpio.PinIO, levels chan<- gpio.Level) {
		pin.WaitForEdge(-1)
		select {
		case <-ctx.Done():
		case levels <- pin.Read():
		}
	}(inst.pin, levels)
	return levels
}
