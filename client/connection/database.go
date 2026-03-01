package connection

import (
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	havenCrypto "haven/client/crypto"
	"haven/client/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OpenDatabase opens the SQLCipher-encrypted client database.
// The key is derived from the Ed25519 private key via HKDF.
func OpenDatabase(privKey ed25519.PrivateKey, dataDir string) (*gorm.DB, error) {
	dbPath := filepath.Join(dataDir, "haven-client.db")

	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	hexKey, err := havenCrypto.DeriveSQLCipherKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("derive sqlcipher key: %w", err)
	}

	dsn := fmt.Sprintf("%s?_pragma_key=x'%s'&_pragma_cipher_page_size=4096", dbPath, hexKey)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	slog.Info("client database opened", "path", dbPath)
	return db, nil
}
