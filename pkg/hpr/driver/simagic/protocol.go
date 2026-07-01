package simagic

// Protocol constants for the Simagic vibration feature report.
// These are private to the driver and intentionally not exported.

const (
	driverName = "simagic"

	stateOff uint8 = 0
	stateOn  uint8 = 1

	// Frame layout (64 bytes total).
	vibrateFrameHeader = 0xF1
	vibrateCommandCode = 0xEC
)

// VID/PID table.
//
// The Simagic family uses Simagic's own VID 0x3670 for most
// products; the P1000 (re-)uses STMicroelectronics' VID 0x0483
// because it ships on an ST evaluation board. Alpha Pedal Neo
// reuses the P500 PID with a different product string.
const (
	vidSimagic uint16 = 0x3670
	vidP1000   uint16 = 0x0483

	pidP500  uint16 = 0x0903
	pidP700  uint16 = 0x0905
	pidP1000 uint16 = 0x0525
	pidP2000 uint16 = 0x0902
)

// isHIDGameController matches the HID usage page/usage for a
// game controller / gamepad. Simagic pedals report themselves
// as such.
func isHIDGameController(usagePage, usage uint16) bool {
	return usagePage == 0x01 && (usage == 0x04 || usage == 0x05)
}

// matchModel inspects the friendly name + VID/PID and returns
// the Simagic Model that owns the device, or ModelUnknown.
//
// The friendly-name check exists because the VID 0x0483 is shared
// with other ST products. We only treat 0x0483/0x0525 as P1000
// when the product string contains "P1000".
func matchModel(vendorID, productID uint16, friendlyName string) Model {
	name := canonicalise(friendlyName)
	hasNameHint := containsAny(name, "P500", "P700", "P1000", "P2000", "ALPHA PEDAL NEO")

	switch {
	case vendorID == vidSimagic && productID == pidP500:
		if !hasNameHint || containsAny(name, "P500", "ALPHA PEDAL NEO") {
			if containsAny(name, "ALPHA PEDAL NEO") {
				return ModelAlphaPedalNeo
			}
			return ModelP500
		}
	case vendorID == vidSimagic && productID == pidP700:
		if !hasNameHint || containsAny(name, "P700") {
			return ModelP700
		}
	case vendorID == vidP1000 && productID == pidP1000:
		if !hasNameHint || containsAny(name, "P1000") {
			return ModelP1000
		}
	case vendorID == vidSimagic && productID == pidP2000:
		if !hasNameHint || containsAny(name, "P2000") {
			return ModelP2000
		}
	}
	return ModelUnknown
}

// vibrateCommand is the wire layout of the feature report. It is
// sent as a single SetFeature call by Device.send.
type vibrateCommand struct {
	FrameHeader uint8
	CommandCode uint8
	Channel     uint8
	State       uint8
	Frequency   uint8
	Amplitude   uint8
	_           [58]byte // pad to 64 bytes
}
