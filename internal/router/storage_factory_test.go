package router

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Vidkin/metrics/internal/config"
)

func TestNewRepository(t *testing.T) {
	tests := []struct {
		name    string
		dbDSN   string
		wantErr bool
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}
