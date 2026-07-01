package hpr

import "errors"

// Sentinel errors returned by the hpr package and conforming drivers.
var (
	// ErrNoDevices is returned by Manager.Scan / Manager.Open when no
	// registered driver matches any device on the system.
	ErrNoDevices = errors.New("hpr: no supported devices found")

	// ErrDeviceClosed is returned by Device methods invoked after Close.
	ErrDeviceClosed = errors.New("hpr: device is closed")

	// ErrUnsupported is returned by a driver when a requested operation
	// is not supported on the underlying device.
	ErrUnsupported = errors.New("hpr: operation not supported by device")
)
