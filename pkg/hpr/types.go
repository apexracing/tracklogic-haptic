// Package hpr contains the vendor-neutral public API used by
// tracklogic-peripherals. See doc.go for an overview.
package hpr

// Target identifies a physical axis on a haptic device
// (clutch / brake / throttle). The set is universal across the
// supported drivers — concrete devices may implement a subset,
// surfaced through the absence of a no-op rather than via flags.
type Target uint8

const (
	TargetClutch   Target = 0
	TargetBrake    Target = 1
	TargetThrottle Target = 2
)

// String returns a human-readable label for the target.
func (t Target) String() string {
	switch t {
	case TargetClutch:
		return "Clutch"
	case TargetBrake:
		return "Brake"
	case TargetThrottle:
		return "Throttle"
	default:
		return "Unknown"
	}
}

// Valid reports whether t is one of the defined Target constants.
func (t Target) Valid() bool {
	return t == TargetClutch || t == TargetBrake || t == TargetThrottle
}

// State is the on/off state of a haptic output.
type State uint8

const (
	Off State = 0
	On  State = 1
)

// Command is a vendor-neutral request to drive a haptic output on a
// given target. Drivers translate the universal frequency/amplitude
// bounds into the device's native representation; out-of-range values
// are clamped silently rather than rejected.
type Command struct {
	Target    Target
	State     State
	Frequency uint8 // 0..50
	Amplitude uint8 // 0..100
}

// DeviceInfo describes a discovered device. It is produced by the
// underlying platform scanner and may be enriched by the claiming
// driver with vendor-specific data (see Model).
type DeviceInfo struct {
	// Model is a driver-specific identifier. Callers that need to
	// interpret it should type-assert to the relevant vendor type
	// (e.g. simagic.Model). The Manager does not inspect it.
	Model any

	// DevicePath is the platform-specific path / identifier used to
	// open the device (e.g. a Windows device interface path).
	DevicePath string

	// FriendlyName is the best human label available, typically
	// "<manufacturer> <product>".
	FriendlyName string

	Manufacturer string
	Product      string

	VendorID      uint16
	ProductID     uint16
	VersionNumber uint32

	UsagePage uint16
	Usage     uint16
}

// ScannedDevice pairs a DeviceInfo with the closure needed to open
// it. The Manager returns these from Scan so callers do not have to
// route through the Manager a second time to open a device.
type ScannedDevice struct {
	Info DeviceInfo
	Open func() (Device, error)
}

// Device is a handle to an open haptic device. Device is not
// goroutine-safe unless the underlying driver documents otherwise;
// the contract is that callers serialise calls (typically by holding
// a single Device value).
type Device interface {
	// Info returns the descriptor of the device (as seen at Scan time).
	Info() DeviceInfo

	// Vibrate sends a command. Drivers MUST clamp Frequency/Amplitude
	// to the supported range; out-of-range values are not an error.
	Vibrate(Command) error

	// Stop turns off the named target. Passing an invalid target is
	// a programming error and returns an error.
	Stop(Target) error

	// StopAll turns off every target on the device. It is called by
	// Close.
	StopAll() error

	// Close releases the device and the underlying transport. It is
	// safe to call more than once; subsequent calls return nil.
	Close() error
}