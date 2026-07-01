//go:build !windows

package hpr

// errUnsupportedPlatform is returned by the default scanner/opener
// on non-Windows builds. The library only ships a Windows backend
// for v1.0; non-Windows callers must provide their own
// DeviceScanner / TransportOpener via WithDeviceScanner /
// WithTransportOpener.
var errUnsupportedPlatform = errPlatformUnsupported{}

// errPlatformUnsupported is exported via errors.go for visibility.
type errPlatformUnsupported struct{}

func (errPlatformUnsupported) Error() string {
	return "hpr: no platform default; supply DeviceScanner and TransportOpener via options"
}

func defaultDeviceScanner() DeviceScanner {
	return unsupportedScanner{}
}

func defaultTransportOpener() TransportOpener {
	return func(DeviceInfo) (Transport, error) {
		return nil, errUnsupportedPlatform
	}
}

type unsupportedScanner struct{}

func (unsupportedScanner) ScanDevices() ([]DeviceInfo, error) {
	return nil, errUnsupportedPlatform
}
