package hpr

type DeviceScanner interface {
	ScanDevices() ([]DeviceInfo, error)
}

type TransportOpener func(DeviceInfo) (Transport, error)

type Manager struct {
	drivers []Driver
	scanner DeviceScanner
	opener  TransportOpener
}

type Option func(*Manager)

func NewManager(options ...Option) *Manager {
	m := &Manager{
		scanner: windowsDeviceScanner{},
		opener:  openHIDTransport,
	}
	for _, option := range options {
		option(m)
	}
	return m
}

func WithDefaultDrivers() Option {
	return func(m *Manager) {
		m.drivers = append(m.drivers, NewSimagicDriver())
	}
}

func WithDrivers(drivers ...Driver) Option {
	return func(m *Manager) {
		m.drivers = append(m.drivers, drivers...)
	}
}

func WithDeviceScanner(scanner DeviceScanner) Option {
	return func(m *Manager) {
		if scanner != nil {
			m.scanner = scanner
		}
	}
}

func WithTransportOpener(opener TransportOpener) Option {
	return func(m *Manager) {
		if opener != nil {
			m.opener = opener
		}
	}
}

func (m *Manager) Scan() ([]DeviceInfo, error) {
	rawDevices, err := m.scanner.ScanDevices()
	if err != nil {
		return nil, err
	}

	devices := make([]DeviceInfo, 0, len(rawDevices))
	for _, info := range rawDevices {
		driver := m.firstMatchingDriver(info)
		if driver == nil {
			continue
		}
		devices = append(devices, m.decorateDeviceInfo(info, driver))
	}
	return devices, nil
}

func (m *Manager) Open(info DeviceInfo) (Device, error) {
	driver := m.driverFor(info)
	if driver == nil {
		return nil, ErrNoPedals
	}

	info = m.decorateDeviceInfo(info, driver)
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

func (m *Manager) OpenFirst() (Device, error) {
	devices, err := m.Scan()
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, ErrNoPedals
	}
	return m.Open(devices[0])
}

func (m *Manager) firstMatchingDriver(info DeviceInfo) Driver {
	for _, driver := range m.drivers {
		if driver.Match(info) {
			return driver
		}
	}
	return nil
}

func (m *Manager) driverFor(info DeviceInfo) Driver {
	if info.DriverName != "" {
		for _, driver := range m.drivers {
			if driver.Name() == info.DriverName && driver.Match(info) {
				return driver
			}
		}
	}
	return m.firstMatchingDriver(info)
}

func (m *Manager) decorateDeviceInfo(info DeviceInfo, driver Driver) DeviceInfo {
	info.DriverName = driver.Name()
	if describer, ok := driver.(interface {
		Describe(DeviceInfo) DeviceInfo
	}); ok {
		info = describer.Describe(info)
		info.DriverName = driver.Name()
	}
	return info
}
