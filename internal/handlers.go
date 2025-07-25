package internal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var db *sql.DB

func init() {
	var err error
	db, err = ConnectToDB()
	if err != nil {
		log.Fatal(err)
	}

	// Проверка подключения
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

var jwtSecret = []byte("")

func Main(c *gin.Context) {
	rows, err := db.Query("SELECT * FROM Products")

	if err != nil {
		c.HTML(http.StatusOK, "index", gin.H{
			"title": "Ginger",
			"user":  nil,
		})
		return
	}
	defer rows.Close()

	var Products []Product

	columns, _ := rows.Columns()

	// Читаем данные
	for rows.Next() {
		// Создаём слоты для данных (все значения как interface{})
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}

		// Сканируем данные в слоты
		err := rows.Scan(pointers...)
		if err != nil {
			log.Fatal(err)
		}
		var product Product
		product.Id = int(values[0].(int64))
		product.Name = string(values[1].([]byte))
		product.Description = string(values[2].([]byte))
		product.Category = string(values[3].([]byte))
		product.Price = int(values[4].(int64))
		product.Creator = string(values[5].([]byte))
		_ = json.Unmarshal(values[6].([]byte), &product.Images)

		Products = append(Products, product)
	}
	log.Print(Products)

	token, err := c.Cookie("jwt-token")
	if err != nil || token == "" {
		c.HTML(http.StatusOK, "index", gin.H{
			"title":    "Ginger",
			"user":     nil,
			"Products": Products,
		})
		return
	}

	email, err := GetEmailFromToken(token)
	if err != nil || email == "" {
		c.HTML(http.StatusOK, "index", gin.H{
			"title":    "Ginger",
			"user":     nil,
			"Products": Products,
		})
		return
	}
	var User User
	err = db.QueryRow("SELECT id, name, email, role FROM Users WHERE email = ?", email).Scan(&User.Id, &User.Name, &User.Email, &User.Role)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Пользователь не найден")
	}

	c.HTML(http.StatusOK, "index", gin.H{
		"title":    "Ginger",
		"user":     User,
		"Products": Products,
	})
}

func CreationPage(c *gin.Context) {
	c.HTML(http.StatusOK, "product-create", "")
}

func Creation(c *gin.Context) {
	var Product Product
	err := c.Bind(&Product)
	if err != nil {
		return
	}
	var paths []string
	form, _ := c.MultipartForm()
	files := form.File["foto"]
	count, _ := GetTotalProducts(db)
	for _, file := range files {
		path := ("/static/product-images/" + fmt.Sprint(count) + "/" + file.Filename)
		paths = append(paths, path)
		// Upload the file to specific dst.
		c.SaveUploadedFile(file, "./web/static/product-images/"+fmt.Sprint(count)+"/"+file.Filename)
	}

	Product.Images = paths

	token, err := c.Cookie("jwt-token")
	if err != nil {
		c.Redirect(http.StatusFound, "")
	}

	email, err := GetEmailFromToken(token)
	if err != nil {
		c.Redirect(http.StatusFound, "")
	}

	var User User
	err = db.QueryRow("SELECT id, name, email, role FROM Users WHERE email = ?", email).Scan(&User.Id, &User.Name, &User.Email, &User.Role)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Пользователь не найден")
	}
	Product.Creator = User.Email

	jsonPaths, _ := json.Marshal(Product.Images)

	result, err := db.Exec("INSERT INTO Products (name, description, category, price, paths, creator) VALUES (?, ?, ?, ?, ?, ?)", Product.Name, Product.Description, Product.Category, Product.Price, jsonPaths, Product.Creator)
	if err != nil {
		log.Fatal(err)
	}
	lastId, err := result.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Inserted row with ID: %d\n", lastId)
	c.Redirect(http.StatusFound, "")
}

func Registration(c *gin.Context) {
	var User User

	if db == nil {
		log.Print("БД ноль")
		c.Redirect(http.StatusFound, "/")
		return
	}

	err := c.Bind(&User)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Не удалось получить данные из формы")
	}
	jsonCart, _ := json.Marshal(User.Cart)
	result, err := db.Exec("INSERT INTO Users (name, email, role, password, cart) VALUES (?, ?, ?, ?, ?)", User.Name, User.Email, User.Role, User.Password, jsonCart)
	if err != nil {
		log.Fatal(err)
	}
	lastId, err := result.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Inserted row with ID: %d\n", lastId)

	token, err := GenerateJWT(User.Email, User.Role)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Не удалось сгенерировать токен")
	}

	c.SetCookie("jwt-token", token, 3600, "/", "localhost", false, true)

	c.Redirect(http.StatusFound, "")
}

func Authorization(c *gin.Context) {
	var User User
	err := c.Bind(&User)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Не удалось получить данные из формы")
	}

	err = db.QueryRow("SELECT id, name, email, role FROM Users WHERE email = ?", User.Email).Scan(&User.Id, &User.Name, &User.Email, &User.Role)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Пользователь не найден")
	}

	token, err := GenerateJWT(User.Email, User.Role)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Не удалось сгенерировать токен")
	}

	c.SetCookie("jwt-token", token, 3600, "/", "localhost", false, true)

	c.Redirect(http.StatusFound, "")
}

func GenerateJWT(email, role string) (string, error) {
	// Время истечения токена (например, 24 часа)
	expirationTime := time.Now().Add(24 * time.Hour)

	// Создаем claims с данными пользователя
	claims := &Claims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "Ginger",
		},
	}

	// Создаем токен с claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен нашим секретным ключом
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GetEmailFromToken(tokenString string) (string, error) {
	// Парсим токен с нашими claims
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем, что используется ожидаемый метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil // Используем тот же секретный ключ, что и при генерации
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	// Извлекаем claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.Email, nil
	}

	return "", fmt.Errorf("invalid token")
}

func Cart(c *gin.Context) {
	c.HTML(http.StatusOK, "cart", gin.H{
		"user": nil,
	})
}
