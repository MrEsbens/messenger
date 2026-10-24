package auth

import (
	"log"
	"net/http"
	"github.com/MrEsbens/messenger-gw/pkg/jwt"
)

func ExtractUserID(r *http.Request, secret []byte) (string, bool) {
	var token string
	cookie, err := r.Cookie("access_token")
	if err == nil && cookie != nil {
		token = cookie.Value
		log.Printf("🔍 Auth: found cookie, len=%d", len(token))
	} else {
		token = r.URL.Query().Get("token")
		log.Printf("🔍 Auth: no cookie. RawQuery: %q | token_len: %d", r.URL.RawQuery, len(token))
	}

	if token == "" {
		log.Println("❌ Auth: token is empty (не передан ?token= в URL)")
		return "", false
	}

	claims, err := jwt.VerifyAccessToken(token, secret)
	if err != nil {
		log.Printf("❌ Auth: JWT verify failed: %v", err)
		return "", false
	}

	log.Printf("✅ Auth: success, user_id=%s", claims.UserID)
	return claims.UserID, true
}