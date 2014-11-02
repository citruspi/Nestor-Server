package main

import (
	"fmt"
	"log"

	"code.google.com/p/go.crypto/bcrypt"
	"github.com/dchest/uniuri"
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
	Tokens   []Token
}

type Token struct {
	Id     int64
	UserId int64
	Token  string
}

func GenerateToken() string {
	return uniuri.NewLenChars(20, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%&()+=-_?"))
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
			var hashedPassword []byte

			hashedPassword, err = bcrypt.GenerateFromPassword([]byte(user.Password), 10)

			if err != nil {
				log.Fatal(err)
			}

			user.Password = string(hashedPassword)

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

func postTokenCollection(c *gin.Context) {
	type Response struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
		Token   string `json:"token,omitempty"`
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

		if count == 0 {
			resp.Success = false
			resp.Error = "User not found."

			c.JSON(404, resp)
		} else {
			var eUser User
			db.Where(&User{Username: user.Username}).First(&eUser)

			err = bcrypt.CompareHashAndPassword([]byte(eUser.Password), []byte(user.Password))

			if err != nil {
				resp.Success = false
				resp.Error = "Incorrect username/password combination."

				c.JSON(403, resp)
			} else {
				contender := GenerateToken()

				for {
					var count int
					db.Model(Token{}).Where("token = ?", contender).Count(&count)

					if count == 0 {
						break
					} else {
						contender = GenerateToken()
					}
				}

				var token Token

				token.Token = contender

				db.Create(&token)

				eUser.Tokens = append(eUser.Tokens, token)

				db.Save(&eUser)

				resp.Success = true
				resp.Token = token.Token

				c.JSON(200, resp)
			}
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
	db.CreateTable(Token{})

	r.POST("/users/", postUserCollection)
	r.POST("/tokens/", postTokenCollection)

	r.Run(":8000")
}
