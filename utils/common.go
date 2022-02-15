package utils

import (
	"github.com/dgrijalva/jwt-go"
	"log"
	"strconv"
)

func GetUser(tokenString string) string {
	type MyCustomClaims struct {
		UserId int `json:"user_id"`
		jwt.StandardClaims
	}
	token, err := jwt.ParseWithClaims(tokenString, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("HS256"), nil
	})
	if err != nil {
		log.Println(err)
	}
	claims, _ := token.Claims.(*MyCustomClaims)
	return strconv.FormatInt(int64(claims.UserId), 10)
}
