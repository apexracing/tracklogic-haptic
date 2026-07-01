package hpr

// Manager is the registry that wires together a set of Driver
// implementations and a platform scanner. Callers build one with
// NewManager and call Scan to enumerate supported devices; opening
// a device goes through the ScannedDevice returned by Scan.
type Manager struct {
	drivers []Driver
}

// Option mutates a Manager during construction.
type Option func(*Manager)

// NewManager constructs a Manager. Drivers are added via WithDrivers;
// the platform scanner is selected at build time (Windows for v1.0).
func NewManager(options ...Option) *Manager {
	m := &Manager{}
	for _, option := range options {
		option(m)
	}
	return m
}

// WithDrivers appends drivers to the manager. Order matters: the
// first driver whose Match returns true for a given device claims it.
func WithDrivers(drivers ...Driver) Option {
	return func(m *Manager) {
		m.drivers = append(m.drivers, drivers...)
	}
}

// scanDevicesImpl is the platform-private device-discovery entry
// point used by Scan. It is normally set by transport_windows.go
// (or by transport_other.go to return a "not supported" error).
// Tests in this package may swap it via the package-private hook
// below so Manager can be exercised without real HID hardware.
var scanDevicesImpl = func() ([]DeviceInfo, error) {
	panic("hpr: no platform default installed; only Windows is supported in v1.0")
}

// Scan enumerates the system for devices claimed by any registered
// driver. The returned slice is filtered: devices no driver matches
// are dropped. Each entry is enriched by the claiming driver's
// Describe (which sets vendor-specific fields such as Model) and
// paired with an Open closure that calls the same driver's Open.
func (m *Manager) Scan() ([]ScannedDevice, error) {
	raw, err := scanDevicesImpl()
	if err != nil {
		return nil, err
	}

	out := make([]ScannedDevice, 0, len(raw))
	for _, info := range raw {
		d, ok := m.claim(info)
		if !ok {
			continue
		}
		info = d.Describe(info)
		driver := d
		out = append(out, ScannedDevice{
			Info: info,
			Open: func() (Device, error) {
				return driver.Open(info)
			},
		})
	}
	return out, nil
}

// claim returns the first driver that matches info.
func (m *Manager) claim(info DeviceInfo) (Driver, bool) {
	for _, d := range m.drivers {
		if d.Match(info) {
			return d, true
		}
	}
	return nil, false
}