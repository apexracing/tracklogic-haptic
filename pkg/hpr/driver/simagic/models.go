// Package simagic implements the hpr.Driver for Simagic haptic
// pedal devices. Supported models are listed in models.go. The
// package knows nothing about the hpr Manager — callers wire it
// in:
//
//	mgr := hpr.NewManager(hpr.WithDrivers(simagic.NewDriver()))
package simagic

// Model identifies a Simagic pedal model. The zero value is
// ModelUnknown; callers should always compare against a defined
// constant.
type Model int

const (
	ModelUnknown Model = iota
	ModelP500
	ModelP700
	ModelP1000
	ModelP2000
	ModelAlphaPedalNeo
)

// String returns the marketing name of the model.
func (m Model) String() string {
	switch m {
	case ModelP500:
		return "Simagic P500"
	case ModelP700:
		return "Simagic P700"
	case ModelP1000:
		return "Simagic P1000"
	case ModelP2000:
		return "Simagic P2000"
	case ModelAlphaPedalNeo:
		return "Simagic Alpha Pedal Neo"
	default:
		return "Simagic Unknown"
	}
}
