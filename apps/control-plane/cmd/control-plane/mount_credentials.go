package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

const mountCredentialModeBasicAuth = "basic-auth"

type signedMountCredentialClaims struct {
	Version   int    `json:"v"`
	NodeID    string `json:"nodeId"`
	MountPath string `json:"mountPath"`
	Username  string `json:"username"`
	Readonly  bool   `json:"readonly"`
	ExpiresAt string `json:"expiresAt"`
}

func issueMountCredential(secret string, nodeID string, mountPath string, readonly bool, issuedAt time.Time, ttl time.Duration) (string, mountCredential, error) {
	credentialID, err := newOpaqueToken()
	if err != nil {
		return "", mountCredential{}, err
	}

	usernameToken, err := newOpaqueToken()
	if err != nil {
		return "", mountCredential{}, err
	}

	claims := signedMountCredentialClaims{
		Version:   1,
		NodeID:    nodeID,
		MountPath: mountPath,
		Username:  "mount-" + usernameToken,
		Readonly:  readonly,
		ExpiresAt: issuedAt.UTC().Add(ttl).Format(time.RFC3339),
	}

	password, err := signMountCredentialClaims(secret, claims)
	if err != nil {
		return "", mountCredential{}, err
	}

	return "mount-" + credentialID, mountCredential{
		Mode:      mountCredentialModeBasicAuth,
		Username:  claims.Username,
		Password:  password,
		ExpiresAt: claims.ExpiresAt,
	}, nil
}

func signMountCredentialClaims(secret string, claims signedMountCredentialClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("encode mount credential claims: %w", err)
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := signMountCredentialPayload(secret, encodedPayload)
	return encodedPayload + "." + signature, nil
}

func signMountCredentialPayload(secret string, encodedPayload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(encodedPayload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
