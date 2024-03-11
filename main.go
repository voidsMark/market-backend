package main

import (
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"golang.org/x/crypto/bcrypt"
)

// TODO: Move to a config or environment variable
var secretKey = []byte("my_freaking_secret_key_true_secret_trust_me")

var (
	db *gorm.DB
)

// Product structure for storing products in the database
type Product struct {
	gorm.Model
	Name   string         `json:"name"`
	Price  float64        `json:"price"`
	Images []ProductImage `json:"images" gorm:"foreignkey:ProductID"`
}

// ProductImage structure for storing product images in the database
type ProductImage struct {
	gorm.Model
	ProductID uint
	URL       string
}

// User structure for storing users in the database
type User struct {
	gorm.Model
	Username string `gorm:"unique;not null"`
	Password string
	Cart     []CartItem `gorm:"foreignkey:UserID"`
}

// CartItem structure for storing cart items
type CartItem struct {
	gorm.Model
	UserID    uint
	ProductID uint
	Quantity  uint
}

// Token structure for storing tokens
type Token struct {
	gorm.Model
	UserID       uint
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

func main() {
	r := gin.Default()

	var err error
	db, err = gorm.Open("postgres", "host=localhost user=postgres dbname=market sslmode=disable password=root")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	db.AutoMigrate(&Product{}, &ProductImage{}, &User{}, &Token{}, &CartItem{})

	r.GET("/products", getProducts)
	r.POST("/products", authMiddleware(), createProduct)

	r.GET("/cart", authMiddleware(), getCart)
	r.POST("/cart/add", authMiddleware(), addToCart)

	r.POST("/register", register)
	r.POST("/login", login)

	// i feeling tired to reset the database by my hands
	// here is endpoint for reseting the database
	r.POST("/reset-database", resetDatabase)

	r.Run("127.0.0.1:8080")
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			log.Println("Empty token")
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return secretKey, nil
		})
		if err != nil || !parsedToken.Valid {
			log.Println("Invalid token: ", parsedToken)
			log.Println("err: ", err)
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Check for token in the database
		var token Token
		if err := db.Where("access_token = ?", tokenString).First(&token).Error; err != nil {
			log.Println("Token not found in the database")
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// check for token expiration
		if token.ExpiresAt < time.Now().Unix() {
			log.Println("Token has expired")
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		c.Set("user_id", parsedToken.Claims.(jwt.MapClaims)["user_id"].(float64))
		c.Next()
	}
}

func getProducts(c *gin.Context) {
	var products []Product
	db.Preload("Images").Find(&products)
	c.JSON(200, products)
}

func createProduct(c *gin.Context) {
	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Example: adding images to the product
	images := []ProductImage{
		{URL: "https://i.ibb.co/qWgscNp/logo.png"},
		{URL: "https://i.ibb.co/qWgscNp/logo.png"},
	}

	product.Images = images

	db.Create(&product)
	c.JSON(201, product)
}

func getCart(c *gin.Context) {
	userID := uint(c.MustGet("user_id").(float64))
	var cartItems []CartItem
	db.Where("user_id = ?", userID).Find(&cartItems)

	// Get product information for each cart item
	var cartProducts []Product
	for _, item := range cartItems {
		var product Product
		db.Preload("Images").First(&product, item.ProductID)
		cartProducts = append(cartProducts, product)
	}

	c.JSON(200, cartProducts)
}

func addToCart(c *gin.Context) {
	userID := uint(c.MustGet("user_id").(float64))
	var input struct {
		ProductID uint `json:"product_id"`
		Quantity  uint `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var existingItem CartItem
	if err := db.Where("user_id = ? AND product_id = ?", userID, input.ProductID).First(&existingItem).Error; err == nil {
		existingItem.Quantity += input.Quantity
		db.Save(&existingItem)
	} else {
		newCartItem := CartItem{
			UserID:    userID,
			ProductID: input.ProductID,
			Quantity:  input.Quantity,
		}
		db.Create(&newCartItem)
	}

	c.JSON(200, gin.H{"message": "Item added to cart successfully"})
}

func register(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Check for unique username
	var existingUser User
	if db.Where("username = ?", user.Username).First(&existingUser).RecordNotFound() {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal Server Error"})
			return
		}
		user.Password = string(hashedPassword)

		db.Create(&user)
		c.JSON(201, gin.H{"message": "User registered successfully"})
	} else {
		c.JSON(400, gin.H{"error": "Username already exists"})
	}
}

func login(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var existingUser User
	if err := db.Where("username = ?", user.Username).First(&existingUser).Error; err != nil {
		c.JSON(401, gin.H{"error": "Invalid credentials"})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(existingUser.Password), []byte(user.Password))
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid credentials"})
		return
	}

	accessToken, refreshToken, err := generateTokens(existingUser.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(200, gin.H{"access_token": accessToken, "refresh_token": refreshToken})
}

func generateTokens(userID uint) (string, string, error) {
	accessToken := jwt.New(jwt.SigningMethodHS256)
	accessToken.Claims = jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Minute * 15).Unix(),
	}
	accessTokenString, err := accessToken.SignedString(secretKey)
	if err != nil {
		return "", "", err
	}

	refreshToken := jwt.New(jwt.SigningMethodHS256)
	refreshToken.Claims = jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
	}
	refreshTokenString, err := refreshToken.SignedString(secretKey)
	if err != nil {
		return "", "", err
	}

	db.Create(&Token{
		UserID:       userID,
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	return accessTokenString, refreshTokenString, nil
}

func resetDatabase(c *gin.Context) {
	// Clear all tables
	db.Exec("DROP TABLE IF EXISTS products, product_images, users, tokens, cart_items")

	// Recreate the tables
	db.AutoMigrate(&Product{}, &ProductImage{}, &User{}, &Token{}, &CartItem{})

	c.JSON(200, gin.H{"message": "Database reset and reinitialized successfully"})
}
