package main

import (
	"context"
	"github.com/onsi/gomega"
	"github.com/sportfun/gakisitor/plugin/plugin_test"
	"periph.io/x/periph/conn/gpio"
	"testing"
	"time"
)

type testingGPIO struct{}

func (*testingGPIO) register(string) error { return nil }
func (*testingGPIO) edge(ctx context.Context) <-chan gpio.Level {
	out := make(chan gpio.Level)
	go func(out chan<- gpio.Level) {
		for {
			out <- gpio.High
			out <- gpio.Low
			time.Sleep(133 * time.Millisecond)
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}(out)

	return out
}

func TestPlugin(t *testing.T) {
	engine.gpio = &testingGPIO{}

	desc := plugin_test.PluginTestDesc{
		ConfigJSON:   `{"gpio":{"pin": "NONE"},"timing":{"clock": "500ms"},"correct": 1}`,
		ValueChecker: gomega.BeNumerically("~", 450, 2),
	}
	plugin_test.PluginValidityChecker(t, &Plugin, desc)
}
