package common

// internal use only to break cycles, use the constants from the kernel package
const (
	TypeEssential    = "essential"
	TypeForeground   = "foreground"
	TypeBackground   = "background"
	StageEssential   = 0
	StageService     = 0x400
	StageApplication = 0x800
)
