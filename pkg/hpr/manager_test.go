package hpr

import (
	"errors"
	"testing"
)

type staticScanner struct {
	devices []DeviceInfo
	err     error
}

func (s staticScanner) ScanDevices() ([]DeviceInfo, error) {
	return s.devices, s.err
}

type fakeDriver struct {
	name     string
	match    bool
	describe func(DeviceInfo) DeviceInfo
	opened   int
}

func (d *fakeDriver) Match(DeviceInfo) bool { return d.match }

func (d *fakeDriver) Describe(info DeviceInfo) DeviceInfo {
	if d.describe != nil {
		return d.describe(info)
	}
	return info
}

func (d *fakeDriver) Open(info DeviceInfo, t Transport) (Device, error) {
	d.opened++
	return stubDevice{info: info, transport: t}, nil
}

func TestManager_ScanFiltersByDriverMatch(t *testing.T) {
	m := NewManager(
		WithDrivers(&fakeDriver{name: "any", match: true}),
		WithDeviceScanner(staticScanner{devices: []DeviceInfo{
			{DevicePath: "a"}, {DevicePath: "b"},
		}}),
	)
	got, err := m.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Scan returned %d devices, want 2", len(got))
	}
}

func TestManager_ScanDropsUnmatchedDevices(t *testing.T) {
	m := NewManager(
		WithDrivers(&fakeDriver{name: "picky", match: false}),
		WithDeviceScanner(staticScanner{devices: []DeviceInfo{{DevicePath: "x"}}}),
	)
	got, err := m.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Scan returned %d devices, want 0", len(got))
	}
}

func TestManager_DriverRegistrationOrderWins(t *testing.T) {
	first := &fakeDriver{name: "first", match: true}
	second := &fakeDriver{name: "second", match: true}
	m := NewManager(
		WithDrivers(first, second),
		WithDeviceScanner(staticScanner{devices: []DeviceInfo{{DevicePath: "x"}}}),
	)
	got, _ := m.Scan()
	if got[0].Info.DevicePath != "x" {
		t.Fatalf("unexpected device: %+v", got[0].Info)
	}
	if first.opened != 0 || second.opened != 0 {
		t.Fatalf("Scan should not open devices, got first=%d second=%d", first.opened, second.opened)
	}
}

func TestManager_ScanCapturesDescribe(t *testing.T) {
	d := &fakeDriver{
		name:  "d",
		match: true,
		describe: func(info DeviceInfo) DeviceInfo {
			info.Model = "decorated"
			return info
		},
	}
	m := NewManager(
		WithDrivers(d),
		WithDeviceScanner(staticScanner{devices: []DeviceInfo{{DevicePath: "x"}}}),
	)
	got, _ := m.Scan()
	if got[0].Info.Model != "decorated" {
		t.Fatalf("Model = %v, want \"decorated\"", got[0].Info.Model)
	}
}

func TestManager_ScannedDeviceOpenUsesClaimedDriver(t *testing.T) {
	d := &fakeDriver{name: "track", match: true}
	// Use a fake opener to avoid touching the real Windows API.
	fakeOpener := func(info DeviceInfo) (Transport, error) {
		return stubTransport{}, nil
	}
	m := NewManager(
		WithDrivers(d),
		WithDeviceScanner(staticScanner{devices: []DeviceInfo{{DevicePath: "x"}}}),
		WithTransportOpener(fakeOpener),
	)
	got, _ := m.Scan()
	dev, err := got[0].Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if dev == nil {
		t.Fatal("Open returned nil device")
	}
	_ = dev.Close()
	if d.opened != 1 {
		t.Fatalf("driver.Open called %d times, want 1", d.opened)
	}
}

func TestManager_ScannedDeviceOpenPropagatesOpenerError(t *testing.T) {
	d := &fakeDriver{name: "x", match: true}
	wantErr := errors.New("opener failed")
	m := NewManager(
		WithDrivers(d),
		WithDeviceScanner(staticScanner{devices: []DeviceInfo{{DevicePath: "x"}}}),
		WithTransportOpener(func(DeviceInfo) (Transport, error) {
			return nil, wantErr
		}),
	)
	got, _ := m.Scan()
	_, err := got[0].Open()
	if !errors.Is(err, wantErr) {
		t.Fatalf("Open: got %v, want %v", err, wantErr)
	}
}

func TestManager_ScanPropagatesScannerError(t *testing.T) {
	wantErr := errors.New("scanner failed")
	m := NewManager(
		WithDrivers(&fakeDriver{name: "any", match: true}),
		WithDeviceScanner(staticScanner{err: wantErr}),
	)
	_, err := m.Scan()
	if !errors.Is(err, wantErr) {
		t.Fatalf("Scan: got %v, want %v", err, wantErr)
	}
}

type stubDevice struct {
	info      DeviceInfo
	transport Transport
}

func (s stubDevice) Info() DeviceInfo        { return s.info }
func (s stubDevice) Vibrate(Command) error  { return nil }
func (s stubDevice) Stop(Target) error      { return nil }
func (s stubDevice) StopAll() error         { return nil }
func (s stubDevice) Close() error           { return s.transport.Close() }

type stubTransport struct{}

func (stubTransport) SetFeature([]byte) error { return nil }
func (stubTransport) Close() error           { return nil }