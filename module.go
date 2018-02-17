package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/sportfun/gakisitor/config"
	"github.com/sportfun/gakisitor/log"
	"gopkg.in/sportfun/gakisitor.v0/env"
	"gopkg.in/sportfun/gakisitor.v0/module"
)

type moduleImpl struct {
	notifications *module.NotificationQueue
	logger        log.Logger
	state         byte

	engine *rpm
}

var (
	debugModuleStarted    = log.NewArgumentBinder("module '%s' started")
	debugModuleConfigured = log.NewArgumentBinder("module '%s' configured")
	debugRpmCalculated    = log.NewArgumentBinder("new rpm calculated")
	debugSessionStarted   = log.NewArgumentBinder("session started")
	debugSessionStopped   = log.NewArgumentBinder("session stopped")
	debugModuleStopped    = log.NewArgumentBinder("module '%s' stopped")
)

func (m *moduleImpl) Start(q *module.NotificationQueue, l log.Logger) error {
	if q == nil {
		m.state = env.PanicState
		return fmt.Errorf("notification queue is not set")
	}
	if l == nil {
		m.state = env.PanicState
		return fmt.Errorf("logger is not set")
	}

	m.logger = l
	m.notifications = q
	m.state = env.StartedState

	l.Debug(debugModuleStarted.Bind(m.Name()))
	return nil
}

func loadConfigurationItem(items map[string]interface{}, name string) (float64, error) {
	_, ok := items[name]
	if !ok {
		return 0, fmt.Errorf("invalid value of '%s' in configuration", name)
	}

	v, ok := items[name].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid value of '%s' in configuration", name)
	}

	return v, nil
}

func (m *moduleImpl) Configure(properties *config.ModuleDefinition) error {
	if properties.Config == nil {
		m.state = env.PanicState
		return fmt.Errorf("configuration needed for this module. RTFM")
	}

	items, ok := properties.Config.(map[string]interface{})
	if !ok {
		m.state = env.PanicState
		return fmt.Errorf("valid configuration needed for this module. RTFM")
	}

	// Check pin configuration
	if pin, err := loadConfigurationItem(items, "rpm.pin"); err != nil {
		m.state = env.PanicState
		return err
	} else {
		m.engine.pin = int(pin)
	}

	// Check buffer configuration
	if buffer, err := loadConfigurationItem(items, "rpm.buffer_time"); err != nil {
		m.state = env.PanicState
		return err
	} else {
		m.engine.bufferTemp = time.Duration(buffer) * time.Millisecond
	}

	// Check clock configuration
	if clock, err := loadConfigurationItem(items, "rpm.refresh_clock"); err != nil {
		m.state = env.PanicState
		return err
	} else {
		m.engine.clock = time.Duration(clock) * time.Millisecond
	}

	m.logger.Debug(debugModuleConfigured.Bind(m.Name()))
	m.state = env.IdleState
	return nil
}

func (m *moduleImpl) Process() error {
	if m.state != env.WorkingState {
		return nil
	}

	if rpm, err := m.engine.get(); err != nil {
		m.logger.Debug(debugRpmCalculated.More("value", rpm))
		m.notifications.NotifyData(m.Name(), "%f", rpm)
	}
	return nil
}

func (m *moduleImpl) Stop() error {
	if m.state == env.WorkingState {
		m.StopSession()
	}

	m.logger.Debug(debugModuleStopped.Bind(m.Name()))
	m.state = env.StoppedState
	return nil
}

func (m *moduleImpl) StartSession() error {
	if m.state == env.WorkingState {
		m.StopSession()
		return fmt.Errorf("session already exist")
	}

	if err := m.engine.start(); err != nil {
		return err
	}
	m.logger.Debug(debugSessionStarted)
	m.state = env.WorkingState
	return nil
}

func (m *moduleImpl) StopSession() error {
	if m.state != env.WorkingState {
		m.state = env.IdleState
		return fmt.Errorf("session not started")
	}

	if err := m.engine.stop(); err != nil {
		return err
	}
	m.logger.Debug(debugSessionStopped)
	m.state = env.IdleState
	return nil
}

func (m *moduleImpl) Name() string { return "rpm.hall" }
func (m *moduleImpl) State() byte  { return m.state }

func ExportModule() module.Module {
	return &moduleImpl{
		engine: &rpm{
			lastRefresh: time.Now(),
			mtx:         sync.Mutex{},
		},
	}
}

// Fix issue #20312 (https://github.com/golang/go/issues/20312)
func main() {}
