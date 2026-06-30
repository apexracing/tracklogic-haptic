package hpr

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const (
	simagicDriverName = "simagic"

	simagicStateOff = 0
	simagicStateOn  = 1

	simagicVibrateFrameHeader = 0xF1
	simagicVibrateCommandCode = 0xEC

	vidSimagic = 0x3670
	vidP1000   = 0x0483
	pidP500    = 0x0903
	pidP700    = 0x0905
	pidP1000   = 0x0525
	pidP2000   = 0x0902
)

type SimagicDriver struct{}

type simagicDevice struct {
	mu        sync.Mutex
	info      DeviceInfo
	transport Transport
	closed    bool
	last      map[Target]normalizedCommand
}

type normalizedCommand struct {
	target    Target
	state     State
	frequency uint8
	amplitude uint8
}

type simagicVibrateCommand struct {
	FrameHeader uint8
	CommandCode uint8
	Channel     uint8
	State       uint8
	Frequency   uint8
	Amplitude   uint8
	_           [58]byte
}

func NewSimagicDriver() Driver {
	return SimagicDriver{}
}

func (SimagicDriver) Name() string {
	return simagicDriverName
}

func (d SimagicDriver) Match(info DeviceInfo) bool {
	return info.isGameController() && d.model(info) != PedalNone
}

func (d SimagicDriver) Describe(info DeviceInfo) DeviceInfo {
	info.Model = d.model(info)
	return info
}

func (d SimagicDriver) Open(info DeviceInfo, transport Transport) (Device, error) {
	info = d.Describe(info)
	device := &simagicDevice{
		info:      info,
		transport: transport,
		last:      make(map[Target]normalizedCommand, 3),
	}
	if err := device.stopAllLocked(true); err != nil {
		return nil, err
	}
	return device, nil
}

func (SimagicDriver) model(info DeviceInfo) PedalModel {
	name := strings.ToUpper(strings.TrimSpace(info.FriendlyName))
	hasModelName := strings.Contains(name, "P500") ||
		strings.Contains(name, "P700") ||
		strings.Contains(name, "P1000") ||
		strings.Contains(name, "P2000") ||
		strings.Contains(name, "ALPHA PEDAL NEO")

	switch {
	case info.VendorID == vidSimagic && info.ProductID == pidP500:
		if !hasModelName || strings.Contains(name, "P500") || strings.Contains(name, "ALPHA PEDAL NEO") {
			return PedalP500
		}
	case info.VendorID == vidSimagic && info.ProductID == pidP700:
		if !hasModelName || strings.Contains(name, "P700") {
			return PedalP700
		}
	case info.VendorID == vidP1000 && info.ProductID == pidP1000:
		if !hasModelName || strings.Contains(name, "P1000") {
			return PedalP1000
		}
	case info.VendorID == vidSimagic && info.ProductID == pidP2000:
		if !hasModelName || strings.Contains(name, "P2000") {
			return PedalP2000
		}
	}
	return PedalNone
}

func (d *simagicDevice) Info() DeviceInfo {
	if d == nil {
		return DeviceInfo{}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.info
}

func (d *simagicDevice) Capabilities() Capabilities {
	return Capabilities{
		Targets:      []Target{TargetClutch, TargetBrake, TargetThrottle},
		MinFrequency: MinFrequency,
		MaxFrequency: MaxFrequency,
		MinAmplitude: MinAmplitude,
		MaxAmplitude: MaxAmplitude,
	}
}

func (d *simagicDevice) Vibrate(command Command) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureOpenLocked(); err != nil {
		return err
	}

	normalized, err := normalizeCommand(command)
	if err != nil {
		return err
	}
	return d.sendLocked(normalized, false)
}

func (d *simagicDevice) Pulse(target Target, frequency, amplitude float32, duration time.Duration) error {
	if err := d.Vibrate(Command{Target: target, State: On, Frequency: frequency, Amplitude: amplitude}); err != nil {
		return err
	}
	if duration <= 0 {
		return nil
	}
	time.Sleep(duration)
	return d.Stop(target)
}

func (d *simagicDevice) Stop(target Target) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureOpenLocked(); err != nil {
		return err
	}
	if !target.Valid() {
		return fmt.Errorf("invalid target: %d", target)
	}
	return d.sendLocked(normalizedCommand{target: target, state: Off}, false)
}

func (d *simagicDevice) StopAll() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureOpenLocked(); err != nil {
		return err
	}
	return d.stopAllLocked(false)
}

func (d *simagicDevice) Close() error {
	if d == nil {
		return nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}

	stopErr := d.stopAllLocked(true)
	closeErr := d.transport.Close()
	d.closed = true
	if closeErr != nil {
		return closeErr
	}
	return stopErr
}

func (d *simagicDevice) ensureOpenLocked() error {
	if d == nil || d.closed || d.transport == nil {
		return ErrDeviceClosed
	}
	return nil
}

func (d *simagicDevice) stopAllLocked(force bool) error {
	var firstErr error
	for _, target := range []Target{TargetClutch, TargetBrake, TargetThrottle} {
		err := d.sendLocked(normalizedCommand{target: target, state: Off}, force)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (d *simagicDevice) sendLocked(command normalizedCommand, force bool) error {
	if !force {
		if last, ok := d.last[command.target]; ok && last == command {
			return nil
		}
	}

	packet := simagicVibrateCommand{
		FrameHeader: simagicVibrateFrameHeader,
		CommandCode: simagicVibrateCommandCode,
		Channel:     uint8(command.target),
		State:       uint8(command.state),
		Frequency:   command.frequency,
		Amplitude:   command.amplitude,
	}
	data := (*[unsafe.Sizeof(simagicVibrateCommand{})]byte)(unsafe.Pointer(&packet))[:]
	if err := d.transport.SetFeature(data); err != nil {
		return err
	}
	d.last[command.target] = command
	return nil
}

func normalizeCommand(command Command) (normalizedCommand, error) {
	if !command.Target.Valid() {
		return normalizedCommand{}, fmt.Errorf("invalid target: %d", command.Target)
	}

	frequency := uint8(clampFloat32(command.Frequency, MinFrequency, MaxFrequency))
	amplitude := uint8(clampFloat32(command.Amplitude, MinAmplitude, MaxAmplitude))
	state := command.State
	if state != On {
		state = Off
	}
	if state == Off || frequency == 0 || amplitude == 0 {
		return normalizedCommand{target: command.Target, state: Off}, nil
	}
	return normalizedCommand{
		target:    command.Target,
		state:     On,
		frequency: frequency,
		amplitude: amplitude,
	}, nil
}

func (i DeviceInfo) isGameController() bool {
	return i.UsagePage == 0x01 && (i.Usage == 0x04 || i.Usage == 0x05)
}
