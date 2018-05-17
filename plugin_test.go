package main

import (
	"github.com/onsi/gomega"
	"github.com/sportfun/gakisitor/plugin/plugin_test"
	"periph.io/x/periph/conn/gpio"
	"testing"
	"time"
)

type testingGPIO struct{}

func (*testingGPIO) register(string) error { return nil }
func (*testingGPIO) egde() <-chan gpio.Level {
	out := make(chan gpio.Level)
	go func(out chan<- gpio.Level) {
		for {
			out <- gpio.High
			out <- gpio.Low
			time.Sleep(50 * time.Millisecond)
		}
	}(out)

	return out
}

func TestPlugin(t *testing.T) {
	_rpm.gpio = &testingGPIO{}

	desc := plugin_test.PluginTestDesc{
		ConfigJSON:   `{"gpio":{"pin": "NONE"},"timing":{"buffer": "1000ms","clock": "250ms"},"correct": 0.97}`,
		ValueChecker: gomega.BeNumerically("~", 1200., 100),
	}
	plugin_test.PluginValidityChecker(t, &Plugin, desc)
}
