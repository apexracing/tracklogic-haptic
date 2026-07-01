// Package hpr defines the vendor-neutral public API for tracklogic-peripherals.
//
// The library targets Windows in v1.0 (HID Raw Input for discovery,
// HidD_SetFeature for transport). On non-Windows platforms the code
// will not compile until a corresponding backend is added; there is
// no runtime "platform not supported" stub because such a stub
// would carry no information.
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
// remember which device came from which driver.
//
// hpr lives under pkg/hpr so that import paths reflect that the
// package is public API surface (per the Go community's pkg/
// convention).
package hpr