package main

import (
	"fmt"
	"net_shop/internal"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// Получаем все названия и расположение html документов
func findFilesWalkDir() ([]string, error) {
	root := "./web/templates"
	pattern := "*.html"

	var paths []string

	// Рекурсивно обходим директорию
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("ошибка доступа к пути %s: %v", path, err)
		}

		// Пропускаем директории
		if d.IsDir() {
			return nil
		}

		// Проверяем совпадение с шаблоном
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return fmt.Errorf("ошибка проверки шаблона для %s: %v", path, err)
		}

		if matched {
			// Читаем содержимое файла
			_, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("ошибка чтения файла %s: %v", path, err)
			}

			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка обхода директории: %v", err)
	}
	return paths, nil
}

func main() {
	router := gin.Default()

	// Получаем пути ко всем html
	paths, _ := findFilesWalkDir()
	router.LoadHTMLFiles(paths...)

	router.Static("/static", "./web/static")

	// Публичные маршруты
	router.POST("/registration", internal.Registration)
	router.POST("/authorization", internal.Authorization)
	router.GET("/", internal.Main)
	router.GET("/product/:id", internal.ShowProduct)
	router.GET("/search", internal.Search)
	router.GET("/products/:category/:subcategory", internal.GetProductsByCategory)

	// Защищенные маршруты (требуется аутентификация)
	authGroup := router.Group("/")
	authGroup.Use(internal.AuthMiddleware()) // Применяем middleware ко всем маршрутам в группе
	{
		authGroup.GET("/cart", internal.Cart)
		authGroup.PATCH("/addtocart", internal.AddToCart)
		authGroup.PATCH("/cart/increase", internal.IncreaseProduct)
		authGroup.PATCH("/cart/decrease", internal.DecreaseProduct)
		authGroup.PATCH("/cart/remove", internal.RemoveProduct)
		authGroup.GET("/profile", internal.Profile)
		authGroup.PATCH("/profile/edit-image", internal.EditProfileImage)
		authGroup.GET("/logout", internal.Logout)
		authGroup.POST("/add-comment", internal.AddComment)
		// Защищённые маршруты (требуется роль продавца)
		salersGroup := authGroup.Group("/")        // Используем authGroup как родительскую
		salersGroup.Use(internal.RoleMiddleware()) // Добавляем middleware проверки роли
		{
			salersGroup.GET("/create-product", internal.CreationPage)
			salersGroup.POST("/creation", internal.Creation)
			salersGroup.PUT("/update-product", internal.UpdateProduct)
			salersGroup.DELETE("/delete-product/:id", internal.DeleteProduct)
		}

	}

	router.Run(":8080")
}
