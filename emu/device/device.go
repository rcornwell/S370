package device

// Interface for devices to handle commands
type Device interface {
	StartIO() uint8
	StartCmd(cmd uint8) uint8
	HaltIO() uint8
	InitDev() uint8
}
