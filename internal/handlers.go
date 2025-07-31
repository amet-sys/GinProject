package internal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var db *sql.DB

func init() {
	var err error
	//Инициализация переменных окружения
	err = godotenv.Load("./internal/secrets.env")
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}
	//Инициаизация базы данных

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
	Products := GetAllProducts()
	if Products == nil {
		token, err := c.Cookie("jwt-token")
		if err != nil || token == "" {
			c.HTML(http.StatusOK, "index", gin.H{
				"title":    "Ginger",
				"user":     nil,
				"Products": nil,
			})
			return
		}

		email, err := GetEmailFromToken(token)
		if err != nil || email == "" {
			c.HTML(http.StatusOK, "index", gin.H{
				"title":    "Ginger",
				"user":     nil,
				"Products": nil,
			})
			return
		}
	}

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
	User.Image = []byte("/static/default-user-image/default.jpg")
	jsonImage, _ := json.Marshal(User.Image)
	result, err := db.Exec("INSERT INTO Users (image, name, email, role, password, cart) VALUES (?, ?, ?, ?, ?, ?)", jsonImage, User.Name, User.Email, User.Role, User.Password, jsonCart)
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

func Profile(c *gin.Context) {
	token, err := c.Cookie("jwt-token")
	if err != nil || token == "" {
		c.Redirect(http.StatusFound, "/")
		return
	}

	email, err := GetEmailFromToken(token)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	var User User
	var jsonImage []byte
	err = db.QueryRow("SELECT id, image, name, email, role FROM Users WHERE email = ?", email).Scan(&User.Id, &jsonImage, &User.Name, &User.Email, &User.Role)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Пользователь не найден")
	}
	err = json.Unmarshal(jsonImage, &User.Image)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	if User.Role != "salesman" {
		c.HTML(http.StatusOK, "profile", gin.H{
			"user":         User,
			"profileImage": User.ToString(),
		})
		return
	}
	var products []Product

	rows, err := db.Query("SELECT id, name, category, subcategory, price, paths FROM Products WHERE creator = ?", email)
	if err != nil {
		log.Printf("Ошибка запроса: %v", err)
		c.HTML(http.StatusOK, "profile", gin.H{
			"user":         User,
			"profileImage": User.ToString(),
		})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var product Product
		var price float32
		var imagesJSON []byte

		err := rows.Scan(
			&product.Id,
			&product.Name,
			&product.Category,
			&product.Subcategory,
			&price,
			&imagesJSON,
		)
		if err != nil {
			log.Printf("Ошибка сканирования: %v", err)
			continue
		}

		product.Price = float64(price)

		if err := json.Unmarshal(imagesJSON, &product.Images); err != nil {
			log.Printf("Ошибка парсинга JSON: %v", err)
			continue
		}

		products = append(products, product)
	}

	// Проверяем ошибки после цикла
	if err := rows.Err(); err != nil {
		log.Printf("Ошибка при обработке результатов: %v", err)
	}

	c.HTML(http.StatusOK, "profile", gin.H{
		"user":         User,
		"profileImage": User.ToString(),
		"products":     products,
	})
}

func EditProfileImage(c *gin.Context) {
	token, err := c.Cookie("jwt-token")
	if err != nil || token == "" {
		c.Redirect(http.StatusFound, "/profile")
		return
	}

	email, err := GetEmailFromToken(token)
	if err != nil {
		c.Redirect(http.StatusFound, "/profile")
		return
	}

	// Получаем файл из формы
	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не удалось получить файл"})
		return
	}

	// Проверяем тип файла
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл должен быть изображением"})
		return
	}
	_ = c.SaveUploadedFile(file, "./web/static/user-images/"+fmt.Sprint(email)+"/"+file.Filename)

	path := []byte("/static/user-images/" + fmt.Sprint(email) + "/" + file.Filename)
	jsonPath, _ := json.Marshal(path)
	_, err = db.Exec("UPDATE Users SET image = ? WHERE email = ?", jsonPath, email)
	if err == nil {
		c.Redirect(http.StatusSeeOther, "/profile")
		return
	}
}

func Logout(c *gin.Context) {
	token, err := c.Cookie("jwt-token")
	if err != nil {
		log.Print("Ошибка в получении токена при попытке выхода с аккаунта")
		c.Redirect(http.StatusFound, "/")
	}
	c.SetCookie("jwt-token", token, -1, "/", "localhost", false, true)
	c.Redirect(http.StatusFound, "/")
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

func AddToCart(c *gin.Context) {
	var req RequestWithProductID
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный запрос"})
		return
	}

	token, _ := c.Cookie("jwt-token")

	email, _ := GetEmailFromToken(token)

	var User User
	var jsonCart []byte
	err := db.QueryRow("SELECT id, name, email, role, cart FROM Users WHERE email = ?", email).Scan(&User.Id, &User.Name, &User.Email, &User.Role, &jsonCart)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Пользователь не найден")
	}
	_ = json.Unmarshal(jsonCart, &User.Cart)
	if User.Cart == nil {
		// Если Cart был nil или JSON пустой, инициализируем новую map
		User.Cart = make(map[string]int)
	}
	// Проверяем наличие товара в корзине
	if _, exists := User.Cart[req.ProductID]; exists {
		User.Cart[req.ProductID] += 1 // Увеличиваем количество, если товар уже есть
	} else {
		User.Cart[req.ProductID] = 1 // Добавляем товар, если его нет
	}
	jsonCart, _ = json.Marshal(User.Cart)
	log.Println(jsonCart)
	log.Println(User.Cart)

	_, err = db.Exec("UPDATE Users SET cart = ? WHERE email = ?", jsonCart, User.Email)
	if err == nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
}

func Cart(c *gin.Context) {
	token, _ := c.Cookie("jwt-token")

	email, _ := GetEmailFromToken(token)

	var User User
	var jsonCart []byte
	err := db.QueryRow("SELECT id, name, email, role, cart FROM Users WHERE email = ?", email).Scan(&User.Id, &User.Name, &User.Email, &User.Role, &jsonCart)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Пользователь не найден")
	}
	_ = json.Unmarshal(jsonCart, &User.Cart)

	var Products []Product
	var total float64
	for i, _ := range User.Cart {
		var product Product
		var paths string
		err := db.QueryRow("SELECT name, price, paths FROM Products WHERE id = ?", i).Scan(&product.Name, &product.Price, &paths)
		if err != nil {
			c.JSON(http.StatusBadGateway, "Продукт")
			return
		}
		_ = json.Unmarshal([]byte(paths), &product.Images)
		product.Counts = User.Cart[i]
		product.Id = i
		total += product.Price * float64(product.Counts)
		Products = append(Products, product)
	}
	c.HTML(http.StatusOK, "cart", gin.H{
		"user":     User,
		"products": Products,
		"total":    total,
	})
}

func DecreaseProduct(c *gin.Context) {
	var req RequestWithProductID
	err := c.ShouldBindJSON(&req)
	log.Print(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, _ := c.Cookie("jwt-token")

	email, _ := GetEmailFromToken(token)

	var User User
	var jsonCart []byte
	err = db.QueryRow("SELECT id, name, email, role, cart FROM Users WHERE email = ?", email).Scan(&User.Id, &User.Name, &User.Email, &User.Role, &jsonCart)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Пользователь не найден")
	}
	_ = json.Unmarshal(jsonCart, &User.Cart)
	if cnt := User.Cart[req.ProductID] - 1; cnt == 0 {
		delete(User.Cart, req.ProductID)
	} else {
		User.Cart[req.ProductID] -= 1
	}
	jsonCart, _ = json.Marshal(User.Cart)

	_, err = db.Exec("UPDATE Users SET cart = ? WHERE email = ?", jsonCart, User.Email)
	if err == nil {
		c.Redirect(http.StatusSeeOther, "/cart")
		return
	}
}
func IncreaseProduct(c *gin.Context) {
	var req RequestWithProductID
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, _ := c.Cookie("jwt-token")

	email, _ := GetEmailFromToken(token)

	var User User
	var jsonCart []byte
	err = db.QueryRow("SELECT id, name, email, role, cart FROM Users WHERE email = ?", email).Scan(&User.Id, &User.Name, &User.Email, &User.Role, &jsonCart)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Пользователь не найден")
	}
	_ = json.Unmarshal(jsonCart, &User.Cart)
	User.Cart[req.ProductID] += 1
	jsonCart, _ = json.Marshal(User.Cart)
	_, err = db.Exec("UPDATE Users SET cart = ? WHERE email = ?", jsonCart, User.Email)
	if err == nil {
		c.Redirect(http.StatusSeeOther, "/cart")
		return
	}
}

func RemoveProduct(c *gin.Context) {
	var req RequestWithProductID
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, _ := c.Cookie("jwt-token")

	email, _ := GetEmailFromToken(token)

	var User User
	var jsonCart []byte
	err = db.QueryRow("SELECT id, name, email, role, cart FROM Users WHERE email = ?", email).Scan(&User.Id, &User.Name, &User.Email, &User.Role, &jsonCart)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Пользователь не найден")
	}
	_ = json.Unmarshal(jsonCart, &User.Cart)
	delete(User.Cart, req.ProductID)
	jsonCart, _ = json.Marshal(User.Cart)
	_, err = db.Exec("UPDATE Users SET cart = ? WHERE email = ?", jsonCart, User.Email)
	if err == nil {
		c.Redirect(http.StatusSeeOther, "/cart")
		return
	}
}

func Search(c *gin.Context) {
	query := c.Query("q")
	var results []Product
	Products := GetAllProducts()
	for _, p := range Products {
		// Простой поиск по названию (регистронезависимый)
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(query)) {
			results = append(results, p)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"results": results,
	})
}

func GetAllProducts() []Product {
	rows, err := db.Query("SELECT * FROM Products")

	if err != nil {
		return nil
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
		product.Id = string(values[0].([]uint8))
		product.Name = string(values[1].([]byte))
		product.Description = string(values[2].([]byte))
		product.Category = string(values[3].([]byte))
		product.Subcategory = string(values[4].([]byte))
		product.Price = float64(values[5].(float32))
		product.Creator = string(values[6].([]byte))
		_ = json.Unmarshal(values[7].([]byte), &product.Images)

		Products = append(Products, product)
	}
	return Products
}

func capitalize(str string) string {
	if str == "" {
		return str
	}

	// Получаем первый руна (Unicode символ)
	r, size := utf8.DecodeRuneInString(str)
	return string(unicode.ToUpper(r)) + str[size:]
}

func GetProductsByCategory(c *gin.Context) {
	category := c.Param("category")       // "produkty-pitaniya"
	subcategory := c.Param("subcategory") // "molochnye-produkty"

	// Преобразуем обратно в читаемый формат
	readableCategory := strings.ReplaceAll(category, "-", " ")
	readableSubcategory := strings.ReplaceAll(subcategory, "-", " ")

	// Применяем capitalize
	readableCategory = capitalize(readableCategory)
	readableSubcategory = capitalize(readableSubcategory)

	Products := GetAllProducts()
	var results []Product

	for _, p := range Products {

		if p.Category == readableCategory && p.Subcategory == readableSubcategory {
			results = append(results, p)
		}
	}

	token, _ := c.Cookie("jwt-token")

	email, _ := GetEmailFromToken(token)

	var User User
	var jsonCart []byte
	err := db.QueryRow("SELECT id, name, email, role, cart FROM Users WHERE email = ?", email).Scan(&User.Id, &User.Name, &User.Email, &User.Role, &jsonCart)
	if err != nil {
		c.HTML(http.StatusOK, "productsbycategory", gin.H{
			"user":     nil,
			"Products": results,
		})
		return
	}

	c.HTML(http.StatusOK, "productsbycategory", gin.H{
		"user":     User,
		"Products": results,
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
	Product.Creator = email

	jsonPaths, _ := json.Marshal(Product.Images)

	result, err := db.Exec("INSERT INTO Products (name, description, category, subcategory, price, paths, creator) VALUES (?, ?, ?, ?, ?, ?, ?)", Product.Name, Product.Description, Product.Category, Product.Subcategory, Product.Price, jsonPaths, Product.Creator)
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

func ShowProduct(c *gin.Context) {
	id := c.Param("id")

	var product Product

	product.Id = id

	row := db.QueryRow(`
        SELECT name, description, category, subcategory, price, paths, creator 
        FROM Products 
        WHERE id = ?`, id)

	var paths string

	err := row.Scan(
		&product.Name,
		&product.Description,
		&product.Category,
		&product.Subcategory,
		&product.Price,
		&paths,
		&product.Creator,
	)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
	}

	_ = json.Unmarshal([]byte(paths), &product.Images)

	token, err := c.Cookie("jwt-token")
	if err != nil || token == "" {
		c.HTML(http.StatusOK, "productPage", gin.H{
			"user":    nil,
			"Product": product,
		})
		return
	}

	email, err := GetEmailFromToken(token)
	if err != nil || email == "" {
		c.HTML(http.StatusOK, "productPage", gin.H{
			"user":    nil,
			"Product": product,
		})
		return
	}
	var User User
	err = db.QueryRow("SELECT id, name, email, role FROM Users WHERE email = ?", email).Scan(&User.Id, &User.Name, &User.Email, &User.Role)
	if err != nil {
		c.JSON(http.StatusBadGateway, "Пользователь не найден")
	}

	c.HTML(http.StatusOK, "productPage", gin.H{
		"user":    User,
		"Product": product,
	})

}

func DeleteProduct(c *gin.Context) {
	id := c.Param("id")
	var Users []User
	rows, err := db.Query(`
    SELECT email, cart 
    FROM Users 
    WHERE JSON_EXTRACT(cart, CONCAT('$."', ?, '"')) IS NOT NULL
`, id)
	if err != nil {
		log.Print("Ошибка во время поиска данных:  ", err)
		c.JSON(http.StatusBadGateway, "Пользователи не найдены")
	}
	defer rows.Close()
	for rows.Next() {
		var user User
		var jsonCart []byte
		if err := rows.Scan(&user.Email, &jsonCart); err != nil {
			log.Fatal("Проблема с базой данных: ", err)
		}
		if err = json.Unmarshal(jsonCart, &user.Cart); err != nil {
			log.Fatal("Ошибка при декодировании: ", err)
		}
		delete(user.Cart, id)
		Users = append(Users, user)
	}
	for _, i := range Users {
		var jsonCart []byte
		jsonCart, err = json.Marshal(i.Cart)
		if err != nil {
			log.Fatal("Ошибка кодирования данных: ", err)
		}
		_, err = db.Exec("UPDATE Users SET cart = ? WHERE email = ?", jsonCart, i.Email)
		if err != nil {
			log.Fatal("Ошибка обновления данных о пользователе: ", err)
		}
	}
	row := db.QueryRow("SELECT paths FROM Products WHERE id = ?", id)
	var paths string
	err = row.Scan(&paths)
	if err != nil {
		log.Fatal("Не удалось извлечь данные из строки", err)
	}
	var images []string
	_ = json.Unmarshal([]byte(paths), &images)
	for _, i := range images {
		log.Print(i)
		err = os.Remove("./web" + i)
		if err != nil {
			log.Fatal("Ошибка удаления файла: ", err)
			return
		}
	}

	_, err = db.Exec("DELETE FROM Products WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusNotFound, "Данного продукта не существует")
	}

	c.Redirect(http.StatusMovedPermanently, "")
}

func UpdateProduct(c *gin.Context) {
	var Product Product
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Print("Ошибка извлечения тела запроса:  ", err)
		return
	}
	defer c.Request.Body.Close()
	_ = json.Unmarshal(body, &Product)
	log.Print(Product)
}
