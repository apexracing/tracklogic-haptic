package hpr

// Driver claims devices and opens them as Device instances. Drivers
// are stateless factories; any per-device state lives on the Device
// returned by Open.
//
// Driver is part of the public API so vendors can ship their own
// implementations under driver/<vendor>/ and register them via
// WithDrivers. The Open method is responsible for constructing the
// device's underlying transport — the Manager does not pass a
// transport in.
type Driver interface {
	// Match reports whether the driver can handle the device. Scan
	// calls Match on every registered driver against every raw
	// scanner result; the first match wins.
	Match(DeviceInfo) bool

	// Describe enriches a raw DeviceInfo with driver-specific
	// fields, typically Model. Drivers that have nothing to add
	// may return info unchanged.
	Describe(DeviceInfo) DeviceInfo

	// Open constructs a Device. The driver owns its transport
	// acquisition and lifecycle: it must Close the transport (on
	// its own, or via Device.Close) if it fails partway through.
	Open(info DeviceInfo) (Device, error)
}