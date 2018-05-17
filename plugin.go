package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sportfun/gakisitor/plugin"
	"github.com/sportfun/gakisitor/profile"
	"strconv"
	"strings"
	"time"
)

var _rpm = rpm{gpio: &_gpio{}}

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
		defer func() { _rpm.close() }()
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
					state = plugin.InSessionState
					_rpm.start()
				case plugin.StopSessionInstruction:
					state = plugin.IdleState
					_rpm.stop()
				}

			case value, open := <-speed:
				if !open {
					continue
				}

				go func() { channels.Data <- value.(float64) * _rpm.correct }()
			}
		}
	},
}

func configure(profile profile.Plugin) (speed <-chan interface{}, err error) {
	var prfGPIO, prfTimingBuffer, prfTimingClock, prfCorrect interface{}

	// prepare GPIO
	for _, prfItem := range []struct {
		path string
		item *interface{}
	}{
		{"gpio.pin", &prfGPIO},
		{"timing.buffer", &prfTimingBuffer},
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
	if bufferSession, err := time.ParseDuration(fmt.Sprint(prfTimingBuffer)); err != nil {
		return nil, errors.WithMessage(err, "timing.buffer")
	} else {
		_rpm.bufferSession = bufferSession
	}
	if refreshClock, err := time.ParseDuration(fmt.Sprint(prfTimingClock)); err != nil {
		return nil, errors.WithMessage(err, "timing.clock")
	} else {
		_rpm.refreshClock = refreshClock
	}
	if correct, err := strconv.ParseFloat(fmt.Sprint(prfCorrect), 64); err != nil {
		return nil, errors.WithMessage(err, "timing.clock")
	} else {
		_rpm.correct = correct
	}
	return _rpm.configure(fmt.Sprint(prfGPIO))
}
