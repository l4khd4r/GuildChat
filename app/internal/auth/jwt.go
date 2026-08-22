package auth

import (
	"crypto/rsa"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)


type JWTManager struct {
	privateKey *rsa.PrivateKey
	publicKey *rsa.PublicKey
	issuer string
	ttl	time.Duration
}


type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func NewJWTManager(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, issuer string, ttl time.Duration) *JWTManager {
	return &JWTManager{
		privateKey: privateKey,
		publicKey: publicKey,
		issuer: issuer,
		ttl: ttl,
	}
}


func stringID(id int64) string {
	return strconv.FormatInt(id, 10)
}


func (manager *JWTManager) GenerateToken(userID int64) (string , error){
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: stringID(userID),
			Issuer: manager.issuer,
			IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(manager.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken , err := token.SignedString(manager.privateKey)
	if err != nil {
		return "" , err
	}

	return signedToken , nil
}

func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodRS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return m.publicKey, nil
		},
	)

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}
