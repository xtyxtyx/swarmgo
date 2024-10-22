package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db   *sql.DB
	once sync.Once
)

func getConnection() *sql.DB {
	once.Do(func() {
		var err error
		db, err = sql.Open("sqlite3", "application.db")
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
	})
	return db
}

func createDatabase() {
	conn := getConnection()

	// Create Users table
	_, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS Users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			first_name TEXT,
			last_name TEXT,
			email TEXT UNIQUE,
			phone TEXT
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create Users table: %v", err)
	}

	// Create PurchaseHistory table
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS PurchaseHistory (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			date_of_purchase TEXT,
			item_id INTEGER,
			amount REAL,
			FOREIGN KEY (user_id) REFERENCES Users(user_id)
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create PurchaseHistory table: %v", err)
	}

	// Create Products table
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS Products (
			product_id INTEGER PRIMARY KEY,
			product_name TEXT NOT NULL,
			price REAL NOT NULL
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create Products table: %v", err)
	}
}

func addUser(userID int, firstName, lastName, email, phone string) {
	conn := getConnection()

	// Check if user already exists
	var exists int
	err := conn.QueryRow("SELECT COUNT(*) FROM Users WHERE user_id = ?", userID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking if user exists: %v", err)
		return
	}
	if exists > 0 {
		return
	}

	_, err = conn.Exec(`
		INSERT INTO Users (user_id, first_name, last_name, email, phone)
		VALUES (?, ?, ?, ?, ?);
	`, userID, firstName, lastName, email, phone)
	if err != nil {
		log.Printf("Error adding user: %v", err)
	}
}

func addPurchase(userID int, dateOfPurchase string, itemID int, amount float64) {
	conn := getConnection()

	// Check if purchase already exists
	var exists int
	err := conn.QueryRow(`
		SELECT COUNT(*) FROM PurchaseHistory
		WHERE user_id = ? AND item_id = ? AND date_of_purchase = ?;
	`, userID, itemID, dateOfPurchase).Scan(&exists)
	if err != nil {
		log.Printf("Error checking if purchase exists: %v", err)
		return
	}
	if exists > 0 {
		return
	}

	_, err = conn.Exec(`
		INSERT INTO PurchaseHistory (user_id, date_of_purchase, item_id, amount)
		VALUES (?, ?, ?, ?);
	`, userID, dateOfPurchase, itemID, amount)
	if err != nil {
		log.Printf("Error adding purchase: %v", err)
	}
}

func addProduct(productID int, productName string, price float64) {
	conn := getConnection()

	_, err := conn.Exec(`
		INSERT INTO Products (product_id, product_name, price)
		VALUES (?, ?, ?);
	`, productID, productName, price)
	if err != nil {
		log.Printf("Error adding product: %v", err)
	}
}

func initializeDatabase() {
	createDatabase()

	// Add initial users
	initialUsers := []struct {
		userID    int
		firstName string
		lastName  string
		email     string
		phone     string
	}{
		{1, "Alice", "Smith", "alice@test.com", "123-456-7890"},
		{2, "Bob", "Johnson", "bob@test.com", "234-567-8901"},
		{3, "Sarah", "Brown", "sarah@test.com", "555-567-8901"},
	}

	for _, user := range initialUsers {
		addUser(user.userID, user.firstName, user.lastName, user.email, user.phone)
	}

	// Add initial purchases
	initialPurchases := []struct {
		userID         int
		dateOfPurchase string
		itemID         int
		amount         float64
	}{
		{1, "2024-01-01", 101, 99.99},
		{2, "2023-12-25", 100, 39.99},
		{3, "2023-11-14", 307, 49.99},
	}

	for _, purchase := range initialPurchases {
		addPurchase(purchase.userID, purchase.dateOfPurchase, purchase.itemID, purchase.amount)
	}

	// Add initial products
	initialProducts := []struct {
		productID   int
		productName string
		price       float64
	}{
		{7, "Hat", 19.99},
		{8, "Wool socks", 29.99},
		{9, "Shoes", 39.99},
	}

	for _, product := range initialProducts {
		addProduct(product.productID, product.productName, product.price)
	}
}

func previewTable(tableName string) {
	conn := getConnection()

	rows, err := conn.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 5;", tableName))
	if err != nil {
		log.Printf("Error querying table %s: %v", tableName, err)
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		log.Printf("Error getting columns for table %s: %v", tableName, err)
		return
	}

	for rows.Next() {
		colsData := make([]interface{}, len(cols))
		colsDataPtrs := make([]interface{}, len(cols))
		for i := range colsData {
			colsDataPtrs[i] = &colsData[i]
		}
		if err := rows.Scan(colsDataPtrs...); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := colsDataPtrs[i].(*interface{})
			rowMap[colName] = *val
		}
		fmt.Println(rowMap)
	}
}
