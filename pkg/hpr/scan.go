package hpr

// Driver claims devices and opens them as Device instances. Drivers
// are stateless factories; any per-device state lives on the Device
// returned by Open.
//
// Driver is part of the public API so that vendors can ship their
// own implementations under driver/<vendor>/ and wire them in with
// WithDrivers. The interface is intentionally narrow: callers of the
// library never interact with a Driver directly — they receive
// ScannedDevice values from Scan and open those.
type Driver interface {
	// Match reports whether the driver can handle the device. Scan
	// calls Match on every registered driver against every raw
	// scanner result; the first match wins.
	Match(DeviceInfo) bool

	// Describe enriches a raw DeviceInfo with driver-specific
	// fields, typically Model. Drivers that have nothing to add
	// may return info unchanged.
	Describe(DeviceInfo) DeviceInfo

	// Open constructs a Device backed by the given transport. The
	// manager-owned closure (see ScannedDevice.Open) creates the
	// transport before calling Open.
	Open(DeviceInfo, Transport) (Device, error)
}

// Transport is the minimal I/O surface a driver needs. On Windows
// the canonical implementation is backed by HidD_SetFeature. Drivers
// that need richer I/O embed this interface and add their own
// methods.
type Transport interface {
	// SetFeature sends a HID feature report.
	SetFeature([]byte) error

	// Close releases the transport. Safe to call more than once.
	Close() error
}

// DeviceScanner enumerates raw devices visible to the OS, before any
// driver filtering. It is the single point of platform-specific
// device discovery in the hpr package.
type DeviceScanner interface {
	ScanDevices() ([]DeviceInfo, error)
}

// TransportOpener creates a Transport for a given DeviceInfo. It is
// the platform-specific counterpart of DeviceScanner.
type TransportOpener func(DeviceInfo) (Transport, error)

// driver is the unexported alias used internally by Manager so it
// can hold drivers without exposing the Driver interface to its
// own struct fields. Driver (the public interface) satisfies driver
// implicitly.
type driver = Driver