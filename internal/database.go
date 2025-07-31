package internal

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func ConnectToDB() (*sql.DB, error) {
	// Формат строки подключения: "username:password@tcp(host:port)/dbname"
	db, err := sql.Open("mysql", os.Getenv("DB_CONFIG"))
	if err != nil {
		log.Printf("Failed to open DB connection: %v", err)
		return nil, err
	}
	// Проверка подключения
	err = db.Ping()
	if err != nil {
		log.Printf("Failed to ping DB: %v", err)
		db.Close() // Закрываем соединение при неудачной проверке
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Products (
		id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
		name VARCHAR(50) NOT NULL,
		description VARCHAR(1000) NOT NULL,
		category VARCHAR(50) NOT NULL,
		subcategory VARCHAR(50) NOT NULL,
		price FLOAT NOT NULL,
		creator VARCHAR(100) NOT NULL,
		paths json NOT NULL);
	`)
	//Если нужно вручную создать БД
	// CREATE TABLE Products (id CHAR(36) PRIMARY KEY DEFAULT (UUID()),name VARCHAR(50) NOT NULL,description VARCHAR(1000) NOT NULL,category VARCHAR(50) NOT NULL, subcategory VARCHAR(50) NOT NULL, price FLOAT NOT NULL,creator VARCHAR(100) NOT NULL,paths JSON NOT NULL);

	if err != nil {
		log.Printf("Failed to create Products table: %v", err)
		db.Close()
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		image MEDIUMBLOB NOT NULL,
		name VARCHAR(50) NOT NULL,
		email VARCHAR(75) NOT NULL,
		role VARCHAR(25) NOT NULL,
		password VARCHAR(1000) NOT NULL,
		cart JSON NOT NULL
	)`)
	//Если нужно вручную создать БД для Users
	//CREATE TABLE Users (id INT AUTO_INCREMENT PRIMARY KEY, image MEDIUMBLOB NOT NULL, name VARCHAR(50) NOT NULL,email VARCHAR(75) NOT NULL,role VARCHAR(25) NOT NULL,password VARCHAR(1000) NOT NULL,cart JSON NOT NULL);

	if err != nil {
		log.Printf("Failed to create Users table: %v", err)
		db.Close()
		return nil, err
	}
	log.Print("Successfully connected to MySQL!")
	return db, nil
}

// Получаем все товары для вывода на главной
func GetTotalProducts(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM Products").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка при подсчете записей: %v", err)
	}
	return count, nil
}
