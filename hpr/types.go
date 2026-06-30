package hpr

import (
	"errors"
	"fmt"
	"time"
)

const (
	MinFrequency = 0
	MaxFrequency = 50
	MinAmplitude = 0
	MaxAmplitude = 100
)

var (
	ErrNoPedals     = errors.New("no supported HPR devices found")
	ErrDeviceClosed = errors.New("HPR device is closed")
)

type Target uint8

const (
	TargetClutch   Target = 0
	TargetBrake    Target = 1
	TargetThrottle Target = 2
)

type Channel = Target

const (
	ChannelClutch   = TargetClutch
	ChannelBrake    = TargetBrake
	ChannelThrottle = TargetThrottle
)

func (t Target) String() string {
	switch t {
	case TargetClutch:
		return "Clutch"
	case TargetBrake:
		return "Brake"
	case TargetThrottle:
		return "Throttle"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}

func (t Target) Valid() bool {
	return t == TargetClutch || t == TargetBrake || t == TargetThrottle
}

type State uint8

const (
	Off State = 0
	On  State = 1
)

type Command struct {
	Target    Target
	State     State
	Frequency float32
	Amplitude float32
}

type Capabilities struct {
	Targets      []Target
	MinFrequency float32
	MaxFrequency float32
	MinAmplitude float32
	MaxAmplitude float32
}

type PedalModel int

const (
	PedalNone PedalModel = iota
	PedalP500
	PedalP700
	PedalP1000
	PedalP2000
)

func (p PedalModel) String() string {
	switch p {
	case PedalP500:
		return "Simagic P500"
	case PedalP700:
		return "Simagic P700"
	case PedalP1000:
		return "Simagic P1000"
	case PedalP2000:
		return "Simagic P2000"
	default:
		return "Unknown"
	}
}

type DeviceInfo struct {
	DriverName    string
	Model         PedalModel
	DevicePath    string
	FriendlyName  string
	Manufacturer  string
	Product       string
	VendorID      uint32
	ProductID     uint32
	VersionNumber uint32
	UsagePage     uint16
	Usage         uint16
}

type PedalInfo = DeviceInfo

func (i DeviceInfo) Open() (Device, error) {
	return NewManager(WithDefaultDrivers()).Open(i)
}

type Driver interface {
	Name() string
	Match(DeviceInfo) bool
	Open(DeviceInfo, Transport) (Device, error)
}

type Device interface {
	Info() DeviceInfo
	Capabilities() Capabilities
	Vibrate(Command) error
	Stop(Target) error
	StopAll() error
	Close() error
}

type Transport interface {
	SetFeature([]byte) error
	Close() error
}

type pulseDevice interface {
	Pulse(Target, float32, float32, time.Duration) error
}

func Pulse(device Device, target Target, frequency, amplitude float32, duration time.Duration) error {
	if p, ok := device.(pulseDevice); ok {
		return p.Pulse(target, frequency, amplitude, duration)
	}
	if err := device.Vibrate(Command{Target: target, State: On, Frequency: frequency, Amplitude: amplitude}); err != nil {
		return err
	}
	if duration > 0 {
		time.Sleep(duration)
	}
	return device.Stop(target)
}

func clampFloat32(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
