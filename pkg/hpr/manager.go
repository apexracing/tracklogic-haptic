package hpr

import "github.com/apexracing/tracklogic-peripherals/internal/hidtransport"

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
// the platform scanner is wired in at init time (Windows HID).
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

// init wires the Windows HID scanner into the Manager.
func init() {
	scanDevices = func() ([]DeviceInfo, error) {
		raw, err := hidtransport.NewScanner().Scan()
		if err != nil {
			return nil, err
		}
		out := make([]DeviceInfo, 0, len(raw))
		for _, d := range raw {
			out = append(out, deviceDescriptorToInfo(d))
		}
		return out, nil
	}
}

// deviceDescriptorToInfo lifts a platform descriptor to the
// universal hpr.DeviceInfo. Model is filled in later by the
// claiming driver's Describe.
func deviceDescriptorToInfo(d hidtransport.DeviceDescriptor) DeviceInfo {
	return DeviceInfo{
		DevicePath:    d.DevicePath,
		FriendlyName:  d.FriendlyName,
		Manufacturer:  d.Manufacturer,
		Product:       d.Product,
		VendorID:      d.VendorID,
		ProductID:     d.ProductID,
		VersionNumber: d.VersionNumber,
		UsagePage:     d.UsagePage,
		Usage:         d.Usage,
	}
}

// scanDevices is the platform-private device-discovery entry point
// used by Scan. It is installed by init() above.
var scanDevices func() ([]DeviceInfo, error)

// Scan enumerates the system for devices claimed by any registered
// driver. The returned slice is filtered: devices no driver matches
// are dropped. Each entry is enriched by the claiming driver's
// Describe (which sets vendor-specific fields such as Model) and
// paired with an Open closure that calls the same driver's Open.
func (m *Manager) Scan() ([]ScannedDevice, error) {
	raw, err := scanDevices()
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
