package internal

import (
	"github.com/golang-jwt/jwt/v5"
)

type Product struct {
	Id          int
	Name        string `form:"name"`
	Description string `form:"description"`
	Category    string `form:"category"`
	Price       int    `form:"price"`
	Images      []string
	Creator     string
}

// func (p *Product) AllPaths() []string {
// 	paths := make([]string, len(p.Images))
// 	for i, img := range p.Images {
// 		paths[i] = string(img)
// 	}
// 	return paths
// }

type User struct {
	Id       int
	Name     string `form:"name"`
	Email    string `form:"email"`
	Password string `form:"password"`
	Role     string `form:"role"`
	Cart     []int
}

// структура для токена
type Claims struct {
	Email    string `json:"email"`
	Password string `json:"password"` // Внимание! Хранение пароля в токене - плохая практика
	Role     string `json:"role"`
	jwt.RegisteredClaims
}
