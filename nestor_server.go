package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

var db gorm.DB
var err error

type User struct {
	Id       int64
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

func postUserCollection(c *gin.Context) {
	type Response struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	var user User
	var resp Response

	c.BindWith(&user, binding.Form)

	if user.Username == "" || user.Password == "" {
		resp.Success = false
		resp.Error = "Incomplete form submission."

		c.JSON(400, resp)
	} else {
		var count int

		db.Model(User{}).Where("username = ?", user.Username).Count(&count)

		fmt.Println(count)

		if count == 0 {
			db.Create(&user)

			resp.Success = true

			c.JSON(200, resp)
		} else {
			resp.Success = false
			resp.Error = fmt.Sprintf("The username '%s' is already in use.", user.Username)

			c.JSON(409, resp)
		}
	}
}

func main() {
	r := gin.Default()

	db, err = gorm.Open("sqlite3", "nestor_server.db")
	if err != nil {
		log.Fatal(err)
	}
	db.DB()
	db.DB().Ping()

	db.CreateTable(User{})

	r.POST("/users/", postUserCollection)

	r.Run(":8000")
}
