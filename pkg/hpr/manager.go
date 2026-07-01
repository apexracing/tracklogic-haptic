package hpr

// Manager is the registry that wires together a DeviceScanner,
// a TransportOpener, and a set of Driver implementations. Callers
// build one with NewManager and call Scan to enumerate supported
// devices; opening a device goes through the ScannedDevice returned
// by Scan, not through Manager.
type Manager struct {
	drivers []driver
	scanner DeviceScanner
	opener  TransportOpener
}

// Option mutates a Manager during construction.
type Option func(*Manager)

// NewManager constructs a Manager with sensible platform defaults:
// on Windows the scanner walks Raw Input devices and the opener
// builds HID transports. Callers then register drivers via
// WithDrivers.
func NewManager(options ...Option) *Manager {
	m := &Manager{}
	for _, option := range options {
		option(m)
	}
	if m.scanner == nil {
		m.scanner = defaultDeviceScanner()
	}
	if m.opener == nil {
		m.opener = defaultTransportOpener()
	}
	return m
}

// WithDrivers appends drivers to the manager. Order matters: the
// first driver whose Match returns true for a given device claims it.
func WithDrivers(drivers ...Driver) Option {
	return func(m *Manager) {
		for _, d := range drivers {
			m.drivers = append(m.drivers, d)
		}
	}
}

// WithDeviceScanner overrides the default device scanner. Useful
// for tests and for callers that want to feed a synthetic device
// list (e.g. a CLI flag for a fixed path).
func WithDeviceScanner(scanner DeviceScanner) Option {
	return func(m *Manager) {
		if scanner != nil {
			m.scanner = scanner
		}
	}
}

// WithTransportOpener overrides the default transport opener. The
// default is platform-appropriate (Windows HID). Override to plug
// in a custom backend, e.g. for a CI test harness.
func WithTransportOpener(opener TransportOpener) Option {
	return func(m *Manager) {
		if opener != nil {
			m.opener = opener
		}
	}
}

// Scan enumerates the system for devices claimed by any registered
// driver. The returned slice is filtered: devices no driver matches
// are dropped. Each entry is enriched by the claiming driver's
// Describe (which sets vendor-specific fields such as Model) and
// paired with an Open closure that captures the right driver and
// transport opener — callers never have to re-resolve the driver.
func (m *Manager) Scan() ([]ScannedDevice, error) {
	if m.scanner == nil {
		return nil, ErrNoDevices
	}
	raw, err := m.scanner.ScanDevices()
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
		opener := m.opener
		out = append(out, ScannedDevice{
			Info: info,
			Open: func() (Device, error) {
				transport, err := opener(info)
				if err != nil {
					return nil, err
				}
				return driver.Open(info, transport)
			},
		})
	}
	return out, nil
}

// claim returns the first driver that matches info.
func (m *Manager) claim(info DeviceInfo) (driver, bool) {
	for _, d := range m.drivers {
		if d.Match(info) {
			return d, true
		}
	}
	return nil, false
}