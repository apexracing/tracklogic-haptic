package simagic

import (
	"errors"
	"testing"

	"github.com/tracklogic/tracklogic-peripherals/pkg/hpr"
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

func p1000Info() hpr.DeviceInfo {
	return hpr.DeviceInfo{
		DevicePath:   "mock",
		FriendlyName: "Simagic P1000",
		VendorID:     vidP1000,
		ProductID:    pidP1000,
		UsagePage:    0x01,
		Usage:        0x04,
	}
}

// newDeviceWith builds a device around a fresh mockTransport. The
// returned mockTransport lets callers inspect what was sent.
func newDeviceWith(t *testing.T) (*device, *mockTransport) {
	t.Helper()
	mt := &mockTransport{}
	dev, err := newDevice(p1000Info(), mt)
	if err != nil {
		t.Fatalf("newDevice: %v", err)
	}
	return dev.(*device), mt
}

func TestDriver_OpenSendsInitialStopAll(t *testing.T) {
	_, mt := newDeviceWith(t)
	if got := len(mt.features); got != 3 {
		t.Fatalf("newDevice should send 3 stop packets, got %d", got)
	}
}

func TestDriver_VibratePacketAndDedup(t *testing.T) {
	dev, mt := newDeviceWith(t)
	cmd := hpr.Command{Target: hpr.TargetBrake, State: hpr.On, Frequency: 25, Amplitude: 80}
	if err := dev.Vibrate(cmd); err != nil {
		t.Fatalf("Vibrate: %v", err)
	}
	if err := dev.Vibrate(cmd); err != nil {
		t.Fatalf("duplicate Vibrate: %v", err)
	}
	if got := len(mt.features); got != 4 {
		t.Fatalf("duplicate command should not send another packet, got %d", got)
	}
	packet := mt.features[3]
	if len(packet) != 64 {
		t.Fatalf("packet length = %d, want 64", len(packet))
	}
	wantPrefix := []byte{vibrateFrameHeader, vibrateCommandCode, byte(hpr.TargetBrake), byte(hpr.On), 25, 80}
	for i, want := range wantPrefix {
		if packet[i] != want {
			t.Fatalf("packet[%d] = 0x%02X, want 0x%02X", i, packet[i], want)
		}
	}
}

func TestDriver_CloseSendsForcedStopAll(t *testing.T) {
	dev, mt := newDeviceWith(t)
	if err := dev.Vibrate(hpr.Command{Target: hpr.TargetBrake, State: hpr.On, Frequency: 10, Amplitude: 20}); err != nil {
		t.Fatalf("Vibrate: %v", err)
	}
	before := len(mt.features)
	if err := dev.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !mt.closed {
		t.Fatal("transport was not closed")
	}
	if got := len(mt.features) - before; got != 3 {
		t.Fatalf("Close should send 3 forced stop packets, got %d", got)
	}
}

func TestDriver_MatchRejectsUnknownVIDPID(t *testing.T) {
	info := hpr.DeviceInfo{VendorID: 0x1111, ProductID: 0x2222, UsagePage: 0x01, Usage: 0x04}
	if NewDriver().Match(info) {
		t.Fatal("unknown VID/PID should not match")
	}
}

func TestDriver_MatchRejectsNonGameController(t *testing.T) {
	info := hpr.DeviceInfo{
		VendorID:     vidP1000,
		ProductID:    pidP1000,
		UsagePage:    0x01,
		Usage:        0x02, // mouse
		FriendlyName: "Simagic P1000",
	}
	if NewDriver().Match(info) {
		t.Fatal("non-game-controller usage should not match")
	}
}

func TestDriver_DescribeSetsModel(t *testing.T) {
	d := NewDriver()
	got := d.Describe(p1000Info())
	if got.Model != ModelP1000 {
		t.Fatalf("Describe Model = %v, want ModelP1000", got.Model)
	}
}

func TestDriver_VibrateAfterCloseReturnsErrDeviceClosed(t *testing.T) {
	dev, _ := newDeviceWith(t)
	_ = dev.Close()
	err := dev.Vibrate(hpr.Command{Target: hpr.TargetBrake, State: hpr.On, Frequency: 1, Amplitude: 1})
	if !errors.Is(err, hpr.ErrDeviceClosed) {
		t.Fatalf("Vibrate after Close: got %v, want ErrDeviceClosed", err)
	}
}

func TestDriver_VibrateClampsFrequencyAndAmplitude(t *testing.T) {
	dev, mt := newDeviceWith(t)
	if err := dev.Vibrate(hpr.Command{
		Target:    hpr.TargetBrake,
		State:     hpr.On,
		Frequency: 200, // above MaxFrequency=50
		Amplitude: 200, // above MaxAmplitude=100
	}); err != nil {
		t.Fatalf("Vibrate: %v", err)
	}
	packet := mt.features[3]
	if packet[4] != MaxFrequency {
		t.Fatalf("frequency on wire = %d, want %d", packet[4], MaxFrequency)
	}
	if packet[5] != MaxAmplitude {
		t.Fatalf("amplitude on wire = %d, want %d", packet[5], MaxAmplitude)
	}
}