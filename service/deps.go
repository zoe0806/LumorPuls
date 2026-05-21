package service

import (
	"lumor_puls/config"

	"gorm.io/gorm"
)

// Deps holds shared service dependencies.
type Deps struct {
	DB     *gorm.DB
	Config config.Config
}
