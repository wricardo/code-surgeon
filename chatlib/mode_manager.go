package chatlib

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/log2"
)

// ModeManager manages the different modes in the chat
type ModeManager struct {
	currentMode IMode
	rwmutex     sync.RWMutex
}

func (mm *ModeManager) BestShot(mode IMode, msg *api.Message) (*api.Message, *api.Command, error) {
	log.Debug().Any("mode", mode).Any("msg", msg).Msg("ModeManager.BestShot started.")
	if msg != nil && strings.HasPrefix(msg.Text, "/") {
		// todo: remove the leading slash up to the first space (inclusive). use regex
		re := regexp.MustCompile(`^/\S*\s?`)
		msg.Text = re.ReplaceAllString(msg.Text, "")

	}
	// No need to lock as we're not accessing mm.currentMode
	return mode.BestShot(msg)
}

func (mm *ModeManager) CreateMode(c *ChatImpl, modeName TMode) (IMode, error) {
	// No need to lock as we're not accessing mm.currentMode
	if constructor, exists := modeRegistry[modeName]; exists {
		return constructor(c), nil
	}
	return nil, fmt.Errorf("unknown mode: %s", modeName)
}

// StartMode starts a new mode
func (mm *ModeManager) StartMode(mode IMode) (*api.Message, *api.Command, error) {
	log2.Debugf("ModeManager.StartMode: %T", mode)
	mm.rwmutex.Lock()
	defer mm.rwmutex.Unlock()

	if mm.currentMode != nil {
		log.Warn().Any("currentMode", mm.currentMode).Any("mode", mode).Msg("start new mode while currentMode != nil")
		if err := mm.stopModeLocked(); err != nil {
			return &api.Message{}, NOOP, err
		}
	}
	mm.currentMode = mode
	res, command, err := mode.Start()
	if command == MODE_QUIT || command == SILENT_MODE_QUIT || command == SILENT_MODE_QUIT_CLEAR {
		if err := mm.stopModeLocked(); err != nil {
			return &api.Message{}, NOOP, err
		}
	}
	return res, command, err
}

// HandleInput handles the user input based on the current mode
func (mm *ModeManager) HandleInput(msg *api.Message) (responseMsg *api.Message, responseCmd *api.Command, err error) {
	defer func() {
		log.Debug().
			Any("responseMsg", responseMsg).
			Any("responseCmd", responseCmd).
			Any("err", err).
			Msg("ModeManager.HandleInput completed.")
	}()
	log.Debug().Any("msg", msg).Msg("ModeManager.HandleInput started.")
	mm.rwmutex.Lock()
	defer mm.rwmutex.Unlock()

	if mm.currentMode != nil {
		response, command, err := mm.currentMode.HandleResponse(msg)
		if command == MODE_QUIT || command == SILENT_MODE_QUIT || command == SILENT_MODE_QUIT_CLEAR {
			if err := mm.stopModeLocked(); err != nil {
				if command == MODE_QUIT {
					return &api.Message{}, NOOP, err
				} else if command == SILENT_MODE_QUIT {
					return &api.Message{}, SILENT, err
				} else if command == SILENT_MODE_QUIT_CLEAR {
					return &api.Message{}, SILENT_CLEAR, err
				} else {
					return &api.Message{}, NOOP, err
				}
			}
		}
		return response, command, err
	}
	return &api.Message{}, NOOP, fmt.Errorf("no mode is currently active")
}

// StopMode stops the current mode
func (mm *ModeManager) StopMode() error {
	mm.rwmutex.Lock()
	defer mm.rwmutex.Unlock()
	return mm.stopModeLocked()
}

// Internal method to stop mode assuming the mutex is already locked
func (mm *ModeManager) stopModeLocked() error {
	if mm.currentMode != nil {
		log.Debug().Any("mode", mm.currentMode.Name()).Msg("ModeManager.stopModeLocked started.")
		log2.Debugf("ModeManager.StopMode: %s", mm.currentMode)
		if err := mm.currentMode.Stop(); err {
			return err
		}
		mm.currentMode = nil
	}
	return nil
}

func (mm *ModeManager) HandleIntent(intent Intent, msg *api.Message) (responseMsg *api.Message, responseCmd *api.Command, err error) {
	log.Debug().Any("mode", intent.TMode).Any("msg", msg).Msg("ModeManager.HandleIntent started.")
	defer func() {
		log.Debug().
			Any("responseMsg", responseMsg).
			Any("responseCmd", responseCmd).
			Any("err", err).
			Msg("ModeManager.HandleIntent completed.")
	}()

	mm.rwmutex.Lock()
	defer mm.rwmutex.Unlock()

	if mm.currentMode != nil {
		if err := mm.stopModeLocked(); err != nil {
			return &api.Message{}, NOOP, err
		}
	}
	mm.currentMode = intent.Mode
	res, command, err := mm.currentMode.HandleIntent(msg, intent)

	if command == MODE_QUIT || command == SILENT_MODE_QUIT || command == SILENT_MODE_QUIT_CLEAR {
		if err := mm.stopModeLocked(); err != nil {
			if command == MODE_QUIT {
				return &api.Message{}, NOOP, err
			} else if command == SILENT_MODE_QUIT {
				return &api.Message{}, SILENT, err
			} else if command == SILENT_MODE_QUIT_CLEAR {
				return &api.Message{}, SILENT_CLEAR, err
			} else {
				return &api.Message{}, NOOP, err
			}
		}
	}
	return res, command, err
}
