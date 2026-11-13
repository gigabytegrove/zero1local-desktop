package main

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const desktopVaultService = "net.gigabytegrove.zero1local.desktop"

// Secrets are stored in Windows Credential Manager, macOS Keychain or the
// Linux desktop Secret Service. Nothing is written into settings.json.
// The Linux secret service must be unlocked and available to this user.
type nativeCredentialVault struct{}

func (nativeCredentialVault) Put(id string, secret []byte) error {
	return keyring.Set(desktopVaultService, id, string(secret))
}
func (nativeCredentialVault) Get(id string) ([]byte, error) {
	value, err := keyring.Get(desktopVaultService, id)
	if err != nil {
		return nil, err
	}
	return []byte(value), nil
}
func (nativeCredentialVault) Delete(id string) error {
	err := keyring.Delete(desktopVaultService, id)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
