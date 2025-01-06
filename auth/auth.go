package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

func New(apiKeyToUserName map[string]string, next http.Handler) *Auth {
	return &Auth{
		Next:             next,
		APIKeyToUserName: apiKeyToUserName,
	}
}

type Auth struct {
	Next             http.Handler
	APIKeyToUserName map[string]string
}

func LoadFromFile(name string) (apiKeyToUserName map[string]string, err error) {
	f, err := os.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m := make(map[string]string)
	if err = json.NewDecoder(f).Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

type userContextKey int

const userKey userContextKey = 0

func GetUser(r *http.Request) (user string, ok bool) {
	user, ok = r.Context().Value(userKey).(string)
	return
}

func (a *Auth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, ok := a.APIKeyToUserName[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	r = r.WithContext(context.WithValue(r.Context(), userKey, user))
	a.Next.ServeHTTP(w, r)
}
