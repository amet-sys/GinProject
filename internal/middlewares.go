package internal

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем токен из куки или заголовка
		tokenString, err := c.Cookie("jwt-token")
		if err != nil {
			tokenString = c.GetHeader("Authorization")
		}

		if tokenString == "" {
			c.Redirect(http.StatusFound, "/") // Перенаправляем на главную
			c.Abort()
			return
		}

		// Проверяем токен
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}

		// Если токен валиден, продолжаем
		c.Next()
	}
}

func RoleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем токен из куки или заголовка
		tokenString, err := c.Cookie("jwt-token")
		if err != nil {
			tokenString = c.GetHeader("Authorization")
		}

		// Парсим токен с нашими claims
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// Проверяем, что используется ожидаемый метод подписи
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil // Используем тот же секретный ключ, что и при генерации
		})

		if err != nil {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}

		// Извлекаем claims
		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			if claims.Role == "salesman" {
				c.Next()
			}
		}

		c.Redirect(http.StatusFound, "/")
		c.Abort()
	}
}
