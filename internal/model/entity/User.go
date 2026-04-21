package entity

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name     string `gorm:"type:varchar(255);not null"`
	Password string `gorm:"type:varchar(255);not null"`
	Age      int
	Gender   string `gorm:"type:varchar(5);null"`
	Contact  string `gorm:"type:varchar(30);null"`
	Email    string `gorm:"type:varchar(255);null;unique"`
}
