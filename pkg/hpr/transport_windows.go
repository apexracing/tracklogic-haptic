//go:build windows

package hpr

import "github.com/tracklogic/tracklogic-peripherals/internal/hidtransport"

func init() {
	scanDevicesImpl = func() ([]DeviceInfo, error) {
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