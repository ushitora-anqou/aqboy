package joypad

type Joypad struct {
	selectAction, selectDirection bool
	action, direction             uint8
}

func NewJoypad() *Joypad {
	return &Joypad{
		selectAction:    true,
		selectDirection: true,
		action:          0x0f,
		direction:       0x0f,
	}
}

func (j *Joypad) Set(val uint8) {
	j.selectAction = ((val >> 5) & 1) == 0
	j.selectDirection = ((val >> 4) & 1) == 0
}

func (j *Joypad) Get() uint8 {
	switch {
	case j.selectAction:
		return j.action
	case j.selectDirection:
		return j.direction
	}
	// FIXME: What behaviour is expected here?
	return 0
}

func (j *Joypad) SetDirection(direction uint8) {
	j.direction = 0x0f &^ direction
}

func (j *Joypad) SetAction(action uint8) {
	j.action = 0x0f &^ action
}
