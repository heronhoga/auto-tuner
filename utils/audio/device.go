package utils

// #include <stdlib.h>
import "C"

import (
	"errors"
	"fmt"

	"github.com/gen2brain/malgo"
)

// replaceDevice keeps the previous device available if the new one cannot start.
// Callers must serialize device changes with shutdown.
func replaceDevice(ctx malgo.Context, current **malgo.Device, config malgo.DeviceConfig, callbacks malgo.DeviceCallbacks, id *malgo.DeviceID) error {
	if id != nil {
		pointer := id.Pointer()
		defer C.free(pointer)
		if config.DeviceType == malgo.Capture {
			config.Capture.DeviceID = pointer
		} else {
			config.Playback.DeviceID = pointer
		}
	}

	next, err := malgo.InitDevice(ctx, config, callbacks)
	if err != nil {
		return err
	}
	previous := *current
	if previous != nil {
		if err := previous.Stop(); err != nil {
			next.Uninit()
			return err
		}
	}
	if err := next.Start(); err != nil {
		next.Uninit()
		if previous != nil {
			if restartErr := previous.Start(); restartErr != nil {
				return errors.Join(err, fmt.Errorf("restore previous device: %w", restartErr))
			}
		}
		return err
	}
	if previous != nil {
		previous.Uninit()
	}
	*current = next
	return nil
}
