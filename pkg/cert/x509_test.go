package cert

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_createAndSave(t *testing.T) {
	type args struct {
		organization      string
		country           string
		pathCertPEM       string
		pathPrivateKeyPEM string
	}
	tests := []struct {
		args    args
		name    string
		wantErr bool
	}{
		{
			name: "create and save ok",
			args: args{
				organization:      "test",
				country:           "US",
				pathCertPEM:       filepath.Join(os.TempDir(), "testCert.PEM"),
				pathPrivateKeyPEM: filepath.Join(os.TempDir(), "testPrivateKey.PEM"),
			},
			wantErr: false,
		},
		{
			name: "create and save bad path cert error",
			args: args{
				organization:      "test",
				country:           "US",
				pathCertPEM:       "/badPath//",
				pathPrivateKeyPEM: filepath.Join(os.TempDir(), "testPrivateKey.PEM"),
			},
			wantErr: true,
		},
		{
			name: "create and save bad path private key error",
			args: args{
				organization:      "test",
				country:           "US",
				pathCertPEM:       filepath.Join(os.TempDir(), "testCert.PEM"),
				pathPrivateKeyPEM: "/badPath//",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				err := createAndSave(tt.args.organization, tt.args.country, tt.args.pathCertPEM, tt.args.pathPrivateKeyPEM)
				assert.NoError(t, err)
				defer os.Remove(tt.args.pathCertPEM)
				defer os.Remove(tt.args.pathPrivateKeyPEM)

				file, err := os.Open(tt.args.pathCertPEM)
				assert.NoError(t, err)
				defer file.Close()
				info, err := file.Stat()
				assert.NoError(t, err)
				assert.NotEqual(t, 0, info.Size())

				file2, err := os.Open(tt.args.pathPrivateKeyPEM)
				assert.NoError(t, err)
				defer file2.Close()
				info, err = file2.Stat()
				assert.NoError(t, err)
				assert.NotEqual(t, 0, info.Size())
			} else {
				err := createAndSave(tt.args.organization, tt.args.country, tt.args.pathCertPEM, tt.args.pathPrivateKeyPEM)
				assert.Error(t, err)
			}
		})
	}
}

func Test_createCertPEM(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "create cert pem",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, certBytes, err := createX509Certificate("test", "country")
			assert.NoError(t, err)

			certPEM, err := createCertPEM(certBytes)
			assert.NoError(t, err)

			assert.NotEqual(t, 0, certPEM.Len())
		})
	}
}

func Test_createPrivateKeyPEM(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "create private key pem",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privateKey, _, err := createX509Certificate("test", "country")
			assert.NoError(t, err)

			privateKeyPEM, err := createPrivateKeyPEM(privateKey)
			assert.NoError(t, err)

			assert.NotNil(t, privateKeyPEM)
		})
	}
}

func Test_createX509Certificate(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "create cert x509 cert",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privateKey, certBytes, err := createX509Certificate("test", "country")
			assert.NoError(t, err)
			assert.NotNil(t, privateKey)
			assert.NotEqual(t, 0, len(certBytes))
		})
	}
}
