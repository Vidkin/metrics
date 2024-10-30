package router

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Vidkin/metrics/internal/config"
)

func TestNewRepository(t *testing.T) {
	tests := []struct {
		cfg             *config.ServerConfig
		name            string
		dbDSN           string
		fileStoragePath string
		wantErr         bool
	}{
		{
			name:    "new postgres storage ok",
			dbDSN:   "user=postgres password=postgres dbname=postgres host=127.0.0.1 port=5432 sslmode=disable",
			wantErr: false,
		},
		{
			name:    "new postgres storage bad server address",
			dbDSN:   "user=postgres password=postgres dbname=postgres host=127.0.0.1 port=99999 sslmode=disable",
			wantErr: true,
		},
		{
			name:            "new file storage ok",
			fileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
			cfg: &config.ServerConfig{
				RetryCount: 1,
				Restore:    true,
			},
			wantErr: false,
		},
		{
			name:            "new file storage bad path",
			fileStoragePath: "/badPath//",
			cfg: &config.ServerConfig{
				RetryCount: 1,
			},
			wantErr: false,
		},
		{
			name:    "new memory storage ok",
			cfg:     &config.ServerConfig{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dbDSN != "" {
				adminDB, err := sql.Open("pgx", tt.dbDSN)
				if err != nil {
					t.Fatalf("Ошибка подключения к БД: %v", err)
				}
				defer adminDB.Close()

				defer func() {
					_, dropErr := adminDB.Exec("DROP TABLE gauge; DROP TABLE counter; DROP TABLE schema_migrations;")
					if dropErr != nil {
						fmt.Printf("Ошибка удаления таблиц БД: %v\n", dropErr)
					}
				}()

				if !tt.wantErr {
					_, err = NewRepository(&config.ServerConfig{DatabaseDSN: tt.dbDSN})
					assert.NoError(t, err)
				} else {
					_, err = NewRepository(&config.ServerConfig{DatabaseDSN: tt.dbDSN})
					assert.Error(t, err)
				}
			} else {
				if tt.fileStoragePath != "" {
					tt.cfg.FileStoragePath = tt.fileStoragePath
				}
				_, err := NewRepository(tt.cfg)

				if !tt.wantErr {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
				}

				os.Remove(tt.fileStoragePath)
			}
		})
	}
}
