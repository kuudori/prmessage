package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"golang.org/x/crypto/pbkdf2"
	_ "modernc.org/sqlite"
)

var xoxcRe = regexp.MustCompile(`xoxc-[a-zA-Z0-9_-]+`)

func slackDataDir() string {
	return filepath.Join(mustHomeDir(), "Library", "Application Support", "Slack")
}

func extractSlackTokens() (token, cookie string, err error) {
	dataDir := slackDataDir()

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return "", "", fmt.Errorf("Slack desktop app not found at %s", dataDir)
	}

	token, err = extractTokenFromLevelDB(filepath.Join(dataDir, "Local Storage", "leveldb"))
	if err != nil {
		return "", "", fmt.Errorf("token extraction failed: %w", err)
	}

	cookie, err = extractCookieFromDB(filepath.Join(dataDir, "Cookies"))
	if err != nil {
		warn("Cookie extraction failed: %v", err)
		warn("You may need to paste the xoxd cookie manually")
	}

	return token, cookie, nil
}

func extractTokenFromLevelDB(dbPath string) (string, error) {
	db, err := leveldb.OpenFile(dbPath, &opt.Options{ReadOnly: true})
	if err != nil {
		return "", fmt.Errorf("cannot open LevelDB (is Slack running? quit it first): %w", err)
	}
	defer db.Close()

	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		val := string(iter.Value())
		if m := xoxcRe.FindString(val); len(m) > 100 {
			return m, nil
		}
	}

	return "", fmt.Errorf("no xoxc token found in Slack local storage")
}

func extractCookieFromDB(dbPath string) (string, error) {
	key, err := getSlackSafeStorageKey()
	if err != nil {
		return "", fmt.Errorf("cannot get Slack Safe Storage key: %w", err)
	}

	derivedKey := pbkdf2.Key([]byte(key), []byte("saltysalt"), 1003, 16, sha1.New)

	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return "", fmt.Errorf("cannot open cookies DB: %w", err)
	}
	defer db.Close()

	var encryptedValue []byte
	err = db.QueryRow("SELECT encrypted_value FROM cookies WHERE name='d' AND host_key='.slack.com'").Scan(&encryptedValue)
	if err != nil {
		return "", fmt.Errorf("cookie 'd' not found: %w", err)
	}

	if len(encryptedValue) < 3 {
		return "", fmt.Errorf("encrypted cookie too short")
	}

	// Strip "v10" prefix (macOS Chrome/Electron format)
	if string(encryptedValue[:3]) == "v10" {
		encryptedValue = encryptedValue[3:]
	}

	decrypted, err := decryptAESCBC(derivedKey, encryptedValue)
	if err != nil {
		return "", fmt.Errorf("cookie decryption failed: %w", err)
	}

	cookie := strings.TrimSpace(string(decrypted))

	if !strings.HasPrefix(cookie, "xoxd-") {
		// Sometimes there's garbage prefix before xoxd-
		idx := strings.Index(cookie, "xoxd-")
		if idx >= 0 {
			cookie = cookie[idx:]
		} else {
			return "", fmt.Errorf("decrypted cookie doesn't contain xoxd token")
		}
	}

	return cookie, nil
}

func getSlackSafeStorageKey() (string, error) {
	out, err := exec.Command("security", "find-generic-password", "-s", "Slack Safe Storage", "-w").Output()
	if err != nil {
		return "", fmt.Errorf("keychain access failed (you may need to allow access): %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}


func decryptAESCBC(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Chrome/Electron uses space-filled IV on macOS
	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = ' '
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext not multiple of block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	if len(plaintext) == 0 {
		return nil, fmt.Errorf("empty plaintext")
	}
	padLen := int(plaintext[len(plaintext)-1])
	if padLen < 1 || padLen > aes.BlockSize {
		return nil, fmt.Errorf("invalid PKCS7 padding value: %d", padLen)
	}
	for i := len(plaintext) - padLen; i < len(plaintext); i++ {
		if plaintext[i] != byte(padLen) {
			return nil, fmt.Errorf("invalid PKCS7 padding")
		}
	}
	plaintext = plaintext[:len(plaintext)-padLen]

	return plaintext, nil
}
