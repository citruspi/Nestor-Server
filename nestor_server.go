package main

import (
	"fmt"
	"log"
	"time"

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
	Id          int64
	Username    string `form:"username" binding:"required"`
	Password    string `form:"password" binding:"required"`
	Tokens      []Token
	Collections []Collection
	Keys        []Key
}

type Token struct {
	Id        int64     `json:"id"`
	UserId    int64     `json:"-"`
	Token     string    `json:"token"`
	Timestamp time.Time `json:"timestamp"`
}

type Collection struct {
	Id     int64  `json:"id"`
	UserId int64  `json:"-"`
	Name   string `form:"name" binding:"required" json:"name"`
	Keys   []Key
}

type Key struct {
	Id           int64
	UserId       int64
	CollectionId int64
	Name         string `form:"name" binding:"required"`
	Key          string `form:"key" binding:"required"`
	Timestamp    time.Time
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
				token.Timestamp = time.Now()

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

func getTokenCollection(c *gin.Context) {
	type Response struct {
		Success bool    `json:"success"`
		Error   string  `json:"error,omitempty"`
		Tokens  []Token `json:"tokens,omitempty"`
	}

	var resp Response

	token := c.Request.FormValue("token")

	if token == "" {
		resp.Success = false
		resp.Error = "Failed to authenticate with a token."

		c.JSON(403, resp)
	} else {
		var count int

		db.Model(Token{}).Where("token = ?", token).Count(&count)

		if count == 0 {
			resp.Success = false
			resp.Error = "Token not found."

			c.JSON(404, resp)
		} else {
			var tokenRecord Token

			db.Model(Token{}).Where("token = ?", token).First(&tokenRecord)

			var tokens []Token

			db.Model(User{}).Where("user_id = ?", tokenRecord.UserId).Find(&tokens)

			resp.Success = true
			resp.Tokens = tokens

			c.JSON(200, resp)
		}
	}
}

func postCollectionCollection(c *gin.Context) {
	type Response struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	var resp Response
	var collection Collection

	token := c.Request.FormValue("token")

	if token == "" {
		resp.Success = false
		resp.Error = "Failed to authenticate with a token."

		c.JSON(403, resp)
	} else {
		var count int

		db.Model(Token{}).Where("token = ?", token).Count(&count)

		if count == 0 {
			resp.Success = false
			resp.Error = "Token not found."

			c.JSON(404, resp)
		} else {

			c.BindWith(&collection, binding.Form)

			if collection.Name == "" {
				resp.Success = false
				resp.Error = "Incomplete form submission."

				c.JSON(400, resp)
			} else {
				var tokenRecord Token

				db.Model(Token{}).Where("token = ?", token).First(&tokenRecord)

				var user User

				db.Model(User{}).Where("id = ?", tokenRecord.UserId).First(&user)

				var collections []Collection

				db.Model(Collection{}).Where("user_id = ?", user.Id).Find(&collections)

				duplicate := false

				for _, item := range collections {
					if item.Name == collection.Name {
						duplicate = true
						break
					}
				}

				if duplicate {
					resp.Success = false
					resp.Error = fmt.Sprintf("A collection named '%s' already exists.", collection.Name)

					c.JSON(409, resp)
				} else {
					db.Create(&collection)

					user.Collections = append(user.Collections, collection)

					db.Save(&user)

					resp.Success = true

					c.JSON(200, resp)
				}
			}
		}
	}
}

func getCollectionCollection(c *gin.Context) {
	type Response struct {
		Success     bool         `json:"success"`
		Error       string       `json:"error,omitempty"`
		Collections []Collection `json:"collections,omitempty"`
	}

	var resp Response

	token := c.Request.FormValue("token")

	if token == "" {
		resp.Success = false
		resp.Error = "Failed to authenticate with a token."

		c.JSON(403, resp)
	} else {
		var count int

		db.Model(Token{}).Where("token = ?", token).Count(&count)

		if count == 0 {
			resp.Success = false
			resp.Error = "Token not found."

			c.JSON(404, resp)
		} else {
			var tokenRecord Token

			db.Model(Token{}).Where("token = ?", token).First(&tokenRecord)

			var collections []Collection

			db.Model(Collection{}).Where("user_id = ?", tokenRecord.UserId).Find(&collections)

			resp.Success = true
			resp.Collections = collections

			c.JSON(200, resp)
		}
	}
}

func postKeyCollection(c *gin.Context) {
	type Response struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	var resp Response
	var key Key

	token := c.Request.FormValue("token")
	collection := c.Request.FormValue("collection")

	if token == "" {
		resp.Success = false
		resp.Error = "Failed to authenticate with a token."

		c.JSON(403, resp)
	} else {
		var count int

		db.Model(Token{}).Where("token = ?", token).Count(&count)

		if count == 0 {
			resp.Success = false
			resp.Error = "Token not found."

			c.JSON(404, resp)
		} else {
			c.BindWith(&key, binding.Form)

			if key.Name == "" || key.Key == "" || collection == "" {
				resp.Success = false
				resp.Error = "Incomplete form submission."

				c.JSON(400, resp)
			} else {
				var tokenRecord Token

				db.Model(Token{}).Where("token = ?", token).First(&tokenRecord)

				var user User

				db.Model(User{}).Where("id = ?", tokenRecord.UserId).First(&user)

				var collectionRecord Collection

				db.Model(Collection{}).Where("id = ?", collection).First(&collectionRecord)

				if collectionRecord.UserId != user.Id {
					resp.Success = false
					resp.Error = "You aren't authorized to access this collection."

					c.JSON(403, resp)
				} else {
					var keys []Key

					db.Model(Key{}).Where("user_id = ?", user.Id).Find(&keys)

					duplicate := false

					for _, item := range keys {
						if item.Name == key.Name {
							duplicate = true
							break
						}
					}

					if duplicate {
						resp.Success = false
						resp.Error = fmt.Sprintf("A key named '%s' already exists.", key.Name)

						c.JSON(409, resp)
					} else {
						key.UserId = user.Id
						key.Timestamp = time.Now()
						db.Create(&key)

						user.Keys = append(user.Keys, key)
						collectionRecord.Keys = append(collectionRecord.Keys, key)

						db.Save(&user)
						db.Save(&collectionRecord)

						resp.Success = true

						c.JSON(200, resp)
					}
				}
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
	db.CreateTable(Collection{})
	db.CreateTable(Key{})

	r.POST("/users/", postUserCollection)
	r.GET("/tokens/", getTokenCollection)
	r.POST("/tokens/", postTokenCollection)
	r.GET("/collections/", getCollectionCollection)
	r.POST("/collections/", postCollectionCollection)
	r.POST("/keys/", postKeyCollection)
	r.Run(":8000")
}
