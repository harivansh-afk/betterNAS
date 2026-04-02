package main

const mountCredentialModeBasicAuth = "basic-auth"

func buildAccountMountCredential(username string) mountCredential {
	return mountCredential{
		Mode:      mountCredentialModeBasicAuth,
		Username:  username,
		Password:  "",
		ExpiresAt: "",
	}
}
