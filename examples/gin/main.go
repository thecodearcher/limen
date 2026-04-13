package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/thecodearcher/limen"
	gormadapter "github.com/thecodearcher/limen/adapters/gorm"
	credentialpassword "github.com/thecodearcher/limen/plugins/credential-password"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("set DATABASE_URL")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	auth, err := limen.New(&limen.Config{
		BaseURL:  "http://localhost:8080",
		Database: gormadapter.New(db),
		Secret:   []byte("0123456789abcdef0123456789abcdef"),
		Plugins: []limen.Plugin{
			credentialpassword.New(),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()

	r.GET("/api/profile", func(c *gin.Context) {
		session, err := auth.GetSession(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user":    session.User,
			"session": session.Session,
		})
	})

	r.Any("/api/auth/*path", func(c *gin.Context) {
		auth.Handler().ServeHTTP(c.Writer, c.Request)
	})

	log.Println("gin example listening on :8080")
	log.Fatal(r.Run(":8080"))
}
