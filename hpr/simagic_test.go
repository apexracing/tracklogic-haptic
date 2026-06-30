package hpr

import (
	"errors"
	"testing"
)

type mockTransport struct {
	features [][]byte
	closed   bool
}

func (m *mockTransport) SetFeature(data []byte) error {
	copied := append([]byte(nil), data...)
	m.features = append(m.features, copied)
	return nil
}

func (m *mockTransport) Close() error {
	m.closed = true
	return nil
}

type staticScanner struct {
	devices []DeviceInfo
	err     error
}

func (s staticScanner) ScanDevices() ([]DeviceInfo, error) {
	return s.devices, s.err
}

type fakeDriver struct {
	name  string
	match bool
}

func (d fakeDriver) Name() string {
	return d.name
}

func (d fakeDriver) Match(DeviceInfo) bool {
	return d.match
}

func (d fakeDriver) Open(info DeviceInfo, transport Transport) (Device, error) {
	return nil, errors.New("not used")
}

func TestSimagicCommandPacketAndDeduplication(t *testing.T) {
	transport := &mockTransport{}
	device, err := NewSimagicDriver().Open(simagicP1000Info(), transport)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if got := len(transport.features); got != 3 {
		t.Fatalf("Open should send initial StopAll packets, got %d packets", got)
	}

	command := Command{Target: TargetBrake, State: On, Frequency: 25, Amplitude: 80}
	if err := device.Vibrate(command); err != nil {
		t.Fatalf("Vibrate failed: %v", err)
	}
	if err := device.Vibrate(command); err != nil {
		t.Fatalf("duplicate Vibrate failed: %v", err)
	}
	if got := len(transport.features); got != 4 {
		t.Fatalf("duplicate command should not send another feature report, got %d packets", got)
	}

	packet := transport.features[3]
	if got := len(packet); got != 64 {
		t.Fatalf("packet length = %d, want 64", got)
	}
	wantPrefix := []byte{0xF1, 0xEC, byte(TargetBrake), byte(On), 25, 80}
	for i, want := range wantPrefix {
		if packet[i] != want {
			t.Fatalf("packet[%d] = 0x%02X, want 0x%02X", i, packet[i], want)
		}
	}
}

func TestSimagicCloseForceSendsStopAllAndClosesTransport(t *testing.T) {
	transport := &mockTransport{}
	device, err := NewSimagicDriver().Open(simagicP1000Info(), transport)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := device.Vibrate(Command{Target: TargetBrake, State: On, Frequency: 10, Amplitude: 20}); err != nil {
		t.Fatalf("Vibrate failed: %v", err)
	}
	beforeClose := len(transport.features)
	if err := device.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if !transport.closed {
		t.Fatal("transport was not closed")
	}
	if got := len(transport.features) - beforeClose; got != 3 {
		t.Fatalf("Close should send three forced stop packets, got %d", got)
	}
}

func TestSimagicDriverDoesNotMatchUnknownDevice(t *testing.T) {
	info := DeviceInfo{
		VendorID:  0x1111,
		ProductID: 0x2222,
		UsagePage: 0x01,
		Usage:     0x04,
	}
	if NewSimagicDriver().Match(info) {
		t.Fatal("unknown VID/PID should not match Simagic driver")
	}
}

func TestManagerDriverRegistrationOrderWins(t *testing.T) {
	manager := NewManager(
		WithDrivers(fakeDriver{name: "first", match: true}, fakeDriver{name: "second", match: true}),
		WithDeviceScanner(staticScanner{devices: []DeviceInfo{{DevicePath: "mock"}}}),
	)

	devices, err := manager.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("Scan returned %d devices, want 1", len(devices))
	}
	if devices[0].DriverName != "first" {
		t.Fatalf("DriverName = %q, want first", devices[0].DriverName)
	}
}

func simagicP1000Info() DeviceInfo {
	return DeviceInfo{
		DevicePath:   "mock",
		FriendlyName: "Simagic P1000",
		VendorID:     vidP1000,
		ProductID:    pidP1000,
		UsagePage:    0x01,
		Usage:        0x04,
	}
}
