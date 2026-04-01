package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type signedMountCredentialClaims struct {
	Version   int    `json:"v"`
	NodeID    string `json:"nodeId"`
	MountPath string `json:"mountPath"`
	Username  string `json:"username"`
	Readonly  bool   `json:"readonly"`
	ExpiresAt string `json:"expiresAt"`
}

func verifyMountCredential(secret string, token string) (signedMountCredentialClaims, error) {
	encodedPayload, signature, ok := strings.Cut(strings.TrimSpace(token), ".")
	if !ok || encodedPayload == "" || signature == "" {
		return signedMountCredentialClaims{}, errors.New("invalid mount credential")
	}

	expectedSignature := signMountCredentialPayload(secret, encodedPayload)
	if subtle.ConstantTimeCompare([]byte(expectedSignature), []byte(signature)) != 1 {
		return signedMountCredentialClaims{}, errors.New("invalid mount credential")
	}

	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return signedMountCredentialClaims{}, errors.New("invalid mount credential")
	}

	var claims signedMountCredentialClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return signedMountCredentialClaims{}, errors.New("invalid mount credential")
	}
	if claims.Version != 1 || claims.NodeID == "" || claims.MountPath == "" || claims.Username == "" || claims.ExpiresAt == "" {
		return signedMountCredentialClaims{}, errors.New("invalid mount credential")
	}

	expiresAt, err := time.Parse(time.RFC3339, claims.ExpiresAt)
	if err != nil || time.Now().UTC().After(expiresAt) {
		return signedMountCredentialClaims{}, errors.New("invalid mount credential")
	}

	return claims, nil
}

func signMountCredentialPayload(secret string, encodedPayload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(encodedPayload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func writeDAVUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="betterNAS"`)
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

func isDAVReadMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, "PROPFIND":
		return true
	default:
		return false
	}
}
