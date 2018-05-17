package main

import (
	"context"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
)

type _gpio struct {
	context.Context
	pin gpio.PinIO
}

func (inst *_gpio) register(pin string) error {
	inst.pin = gpioreg.ByName(pin)
	return inst.pin.In(gpio.PullUp, gpio.RisingEdge)
}

func (inst *_gpio) egde() <-chan gpio.Level {
	levels := make(chan gpio.Level)
	go func(pin gpio.PinIO, levels chan<- gpio.Level) {
		pin.WaitForEdge(-1)
		levels <- pin.Read()
	}(inst.pin, levels)
	return levels
}
