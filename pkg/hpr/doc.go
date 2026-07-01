// Package hpr defines the vendor-neutral public API for tracklogic-peripherals.
//
// The package exposes three things:
//
//   - DeviceInfo / Device / Command: the shape of a haptic device and
//     how to drive it. Callers only see these.
//   - Driver: the extension point. Vendor-specific code lives under
//     driver/<vendor>/ and implements Driver; the platform transport
//     layer lives under internal/.
//   - Manager + Options: the composition root. Build one, register
//     drivers, call Scan.
//
// Typical usage:
//
//	mgr := hpr.NewManager(hpr.WithDrivers(simagic.NewDriver()))
//	devices, err := mgr.Scan()
//	if err != nil || len(devices) == 0 { ... }
//	dev, err := devices[0].Open()
//	if err != nil { ... }
//	defer dev.Close()
//	dev.Vibrate(hpr.Command{
//	    Target:    hpr.TargetBrake,
//	    State:     hpr.On,
//	    Frequency: 30,
//	    Amplitude: 80,
//	})
//
// There is intentionally no Manager.Open: opening goes through the
// ScannedDevice returned by Scan, so the Manager does not need to
// remember which device came from which driver. ScannedDevice.Open
// carries the right Driver for its device.
//
// Transport / device-discovery plumbing is intentionally not part of
// the public API: it is a single Windows HID backend living under
// internal/hidtransport, wired into Manager by build tag. New
// platforms should add their own transport_*.go to this package.
//
// hpr lives under pkg/hpr so that import paths reflect that the
// package is public API surface (per the Go community's pkg/
// convention).
package hpr