package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/gen2brain/malgo"
)

type AudioDeviceSelector interface {
	Devices() ([]malgo.DeviceInfo, error)
	SelectDevice(*malgo.DeviceID) error
}

func NewAudioDevices(input, output AudioDeviceSelector) fyne.CanvasObject {
	return container.NewGridWithColumns(2,
		newDeviceSelect("Audio input", input),
		newDeviceSelect("Audio output", output),
	)
}

func newDeviceSelect(label string, audio AudioDeviceSelector) fyne.CanvasObject {
	options := []string{"System default"}
	ids := map[string]malgo.DeviceID{}
	devices, err := audio.Devices()
	if err != nil {
		return container.NewVBox(widget.NewLabel(label), widget.NewLabel(fmt.Sprintf("Cannot list devices: %v", err)))
	}
	for index, device := range devices {
		name := device.Name()
		if device.IsDefault != 0 {
			name += " (default)"
		}
		// Include the list index so identically named devices remain selectable.
		option := fmt.Sprintf("%d. %s", index+1, name)
		options = append(options, option)
		ids[option] = device.ID
	}

	selectDevice := widget.NewSelect(options, nil)
	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord
	status.Hide()
	selected := options[0]
	selectDevice.SetSelected(selected)
	selectDevice.OnChanged = func(option string) {
		if option == selected {
			return
		}
		var id *malgo.DeviceID
		if value, ok := ids[option]; ok {
			id = &value
		}
		if err := audio.SelectDevice(id); err != nil {
			selectDevice.SetSelected(selected)
			status.SetText(fmt.Sprintf("Could not switch device: %v", err))
			status.Show()
			return
		}
		selected = option
		status.Hide()
	}
	return container.NewVBox(widget.NewLabel(label), selectDevice, status)
}
