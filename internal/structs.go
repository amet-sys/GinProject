package internal

import (
	"github.com/golang-jwt/jwt/v5"
)

type Product struct {
	Id          string   `json:"id"`
	Name        string   `form:"name" json:"name"`
	Description string   `form:"description" json:"description"`
	Category    string   `form:"category" json:"category"`
	Subcategory string   `form:"subcategory" json:"subcategory"`
	Price       float64  `form:"price" json:"price"`
	Images      []string `json:"images"`
	Creator     string   `json:"creator"`
	Counts      int
}

type User struct {
	Id       int
	Image    []byte
	Name     string `form:"name"`
	Email    string `form:"email"`
	Password string `form:"password"`
	Role     string `form:"role"`
	Cart     map[string]int
}

func (u *User) ToString() string {
	return string(u.Image)
}

type RequestWithProductID struct {
	ProductID string `json:"productId"`
}

// структура для токена
type Claims struct {
	Email    string `json:"email"`
	Password string `json:"password"` // Внимание! Хранение пароля в токене - плохая практика
	Role     string `json:"role"`
	jwt.RegisteredClaims
}
