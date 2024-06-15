package device

// Interface for devices to handle commands
type Device interface {
	Start_IO() uint8
	Start_cmd(cmd uint8) uint8
	Halt_IO() uint8
	Init_Dev() uint8
}

//type device struct{}

// func (d *device) Start_IO() uint8 {
// 	return 0
// }

// func (d *device) Start_cmd(cmd uint8) uint8 {
// 	return 0
// }

// func (d *device) Halt_IO() uint8 {
// 	return 0
// }

// func (d *device) Init_Dev() uint8 {
// 	return 0
// }
