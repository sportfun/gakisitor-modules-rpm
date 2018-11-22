package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/sportfun/gakisitor/plugin"
	"github.com/sportfun/gakisitor/profile"
	"strconv"
	"strings"
	"time"
)

var engine = rpm{Context: context.Background(), gpio: &_gpio{}}
var log = logrus.WithField("plugin", "value")

var Plugin = plugin.Plugin{
	Name: "RPM Plugin",
	Instance: func(ctx context.Context, profile profile.Plugin, channels plugin.Chan) error {
		// prepare plugin env
		state := plugin.IdleState
		speed, err := configure(profile)
		if err != nil {
			return errors.WithMessage(err, "configuration failed")
		}

		// process
		defer func() { engine.close() }()
		for {
			select {
			case <-ctx.Done():
				return nil

			case instruction, valid := <-channels.Instruction:
				if !valid {
					return nil
				}

				switch instruction {
				case plugin.StatusPluginInstruction:
					channels.Status <- state
				case plugin.StartSessionInstruction:
					if state == plugin.InSessionState {
						break
					}
					state = plugin.InSessionState
					engine.start()
				case plugin.StopSessionInstruction:
					if state == plugin.IdleState {
						break
					}
					state = plugin.IdleState
					engine.stop()
				}

			case value, open := <-speed:
				if !open {
					continue
				}

				go func() { channels.Data <- value.(float64) * engine.correct }()
			}
		}
	},
}

func configure(profile profile.Plugin) (speed <-chan interface{}, err error) {
	var prfGPIO, prfTimingClock, prfCorrect interface{}

	// prepare GPIO
	for _, prfItem := range []struct {
		path string
		item *interface{}
	}{
		{"gpio.pin", &prfGPIO},
		{"timing.clock", &prfTimingClock},
		{"correct", &prfCorrect},
	} {
		path := make([]interface{}, strings.Count(prfItem.path, ".")+1)
		for i, v := range strings.Split(prfItem.path, ".") {
			path[i] = v
		}

		if *prfItem.item, err = profile.AccessTo(path...); err != nil {
			return nil, errors.WithMessage(err, prfItem.path)
		}
	}

	// configure engine
	if refreshClock, err := time.ParseDuration(fmt.Sprint(prfTimingClock)); err != nil {
		return nil, errors.WithMessage(err, "timing.clock")
	} else {
		engine.ioTick = refreshClock
	}
	if correct, err := strconv.ParseFloat(fmt.Sprint(prfCorrect), 64); err != nil {
		return nil, errors.WithMessage(err, "correct")
	} else {
		engine.correct = correct
	}
	return engine.configure(fmt.Sprint(prfGPIO))
}
