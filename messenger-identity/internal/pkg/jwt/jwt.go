package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	signingKey []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewService(signingKey []byte, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		signingKey: signingKey,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *Service) Generate(userID string) (accessToken, refreshToken string, err error) {
	if userID == "" {
		return "", "", errors.New("userID is required")
	}

	accessToken, err = s.newToken(userID, s.accessTTL)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = s.newToken(userID, s.refreshTTL)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func (s *Service) newToken(userID string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "messenger-identity",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.signingKey)
}

func (s *Service) Verify(token string) (string, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return s.signingKey, nil
	})
	if err != nil || !parsed.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	return claims.UserID, nil
}

func (s *Service) VerifyRefresh(token string) (string, error) {
	return s.Verify(token)
}
