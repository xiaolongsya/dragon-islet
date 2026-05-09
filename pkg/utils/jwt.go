package utils

import (
	"dragon-islet/internal/global"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type MyClaims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint) (string, error) {
	claims := MyClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 1天过期
			Issuer:    "dragon-islet",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(global.CONFIG.GetString("jwt.secret")))
}

func ParseToken(tokenString string) (*MyClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(global.CONFIG.GetString("jwt.secret")), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, err
}
