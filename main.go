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
	//Получам пути ко всем html
	paths, _ := findFilesWalkDir()
	router.LoadHTMLFiles(paths...)

	router.Static("/static", "./web/static")

	router.POST("/registration", internal.Registration)
	router.POST("/authorization", internal.Authorization)

	router.GET("/", internal.Main)
	router.GET("/product/:id")
	router.GET("/cart", internal.Cart)

	router.GET("/create-product", internal.CreationPage)
	router.POST("/creation", internal.Creation)

	router.Run(":8080")
}
