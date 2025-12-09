package models

type RoleUser struct {
	RoleID uint64 `gorm:"primaryKey"`
	UserID uint64 `gorm:"primaryKey"`
}

func (RoleUser) TableName() string {
	return "role_user"
}
