package hpr

// Manager is the registry that wires together DeviceScanner,
// TransportOpener, and a set of Driver implementations. It is the
// single entry point of the public API.
type Manager struct {
	drivers []Driver
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
		m.drivers = append(m.drivers, drivers...)
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
// are dropped. Each entry's DriverName and Model fields are set
// by the claiming driver.
func (m *Manager) Scan() ([]DeviceInfo, error) {
	if m.scanner == nil {
		return nil, ErrNoDevices
	}
	raw, err := m.scanner.ScanDevices()
	if err != nil {
		return nil, err
	}

	out := make([]DeviceInfo, 0, len(raw))
	for _, info := range raw {
		driver := m.firstMatchingDriver(info)
		if driver == nil {
			continue
		}
		out = append(out, m.decorate(info, driver))
	}
	return out, nil
}

// Open opens the device described by info. The driver is resolved
// either by DriverName (when set) or by re-running Match. The
// returned Device owns its Transport; callers close it via
// Device.Close.
func (m *Manager) Open(info DeviceInfo) (Device, error) {
	driver := m.driverFor(info)
	if driver == nil {
		return nil, ErrNoDevices
	}

	info = m.decorate(info, driver)
	transport, err := m.opener(info)
	if err != nil {
		return nil, err
	}
	device, err := driver.Open(info, transport)
	if err != nil {
		_ = transport.Close()
		return nil, err
	}
	return device, nil
}

// OpenFirst scans and opens the first matching device. Useful for
// single-device CLIs and quick scripts.
func (m *Manager) OpenFirst() (Device, error) {
	devices, err := m.Scan()
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, ErrNoDevices
	}
	return m.Open(devices[0])
}

func (m *Manager) firstMatchingDriver(info DeviceInfo) Driver {
	for _, d := range m.drivers {
		if d.Match(info) {
			return d
		}
	}
	return nil
}

func (m *Manager) driverFor(info DeviceInfo) Driver {
	if info.DriverName != "" {
		for _, d := range m.drivers {
			if d.Name() == info.DriverName && d.Match(info) {
				return d
			}
		}
	}
	return m.firstMatchingDriver(info)
}

// decorate runs the optional Describer hook on the driver and
// stamps DriverName. Drivers that need to fill in vendor-specific
// fields (e.g. Model) implement an unexported Describe via type
// assertion below.
func (m *Manager) decorate(info DeviceInfo, driver Driver) DeviceInfo {
	if d, ok := driver.(interface {
		Describe(DeviceInfo) DeviceInfo
	}); ok {
		info = d.Describe(info)
	}
	info.DriverName = driver.Name()
	return info
}
