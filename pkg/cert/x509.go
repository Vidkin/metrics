/*
Package cert provides functionality for creating and saving
X.509 certificates and corresponding private keys in PEM format.
*/
package cert

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"
)

// saveFile saves data to a file at the specified path.
// Parameters:
//   - path: the path to the file where data will be saved.
//   - data: the data to be saved.
//
// Returns an error if the saving fails.
func saveFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0755)
}

// createX509Certificate creates a new X.509 certificate and the corresponding
// private RSA key. The certificate will be valid for the IP addresses
// 127.0.0.1 and ::1, and will have a validity period of 10 years.
// Parameters:
//   - organization: the name of the organization for which the certificate is created.
//   - country: the country where the organization is registered.
//
// Returns the private key, certificate bytes, and an error if the certificate creation fails.
func createX509Certificate(organization, country string) (*rsa.PrivateKey, []byte, error) {
	// создаём шаблон сертификата
	cert := &x509.Certificate{
		// указываем уникальный номер сертификата
		SerialNumber: big.NewInt(1),
		// заполняем базовую информацию о владельце сертификата
		Subject: pkix.Name{
			Organization: []string{organization},
			Country:      []string{country},
		},
		// разрешаем использование сертификата для 127.0.0.1 и ::1
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		// сертификат верен, начиная со времени создания
		NotBefore: time.Now(),
		// время жизни сертификата — 10 лет
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		// устанавливаем использование ключа для цифровой подписи,
		// а также клиентской и серверной авторизации
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	// создаём новый приватный RSA-ключ длиной 4096 бит
	// обратите внимание, что для генерации ключа и сертификата
	// используется rand.Reader в качестве источника случайных данных
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	// создаём сертификат x.509
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, certBytes, nil
}

// createCertPEM encodes the certificate in PEM format.
// Parameters:
//   - certBytes: the bytes of the certificate to encode.
//
// Returns a buffer with the encoded certificate and an error if encoding fails.
func createCertPEM(certBytes []byte) (bytes.Buffer, error) {
	// кодируем сертификат и ключ в формате PEM, который
	// используется для хранения и обмена криптографическими ключами
	var certPEM bytes.Buffer
	err := pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	return certPEM, err
}

// createPrivateKeyPEM encodes the private key in PEM format.
// Parameters:
//   - privateKey: the private RSA key to encode.
//
// Returns a buffer with the encoded private key and an error if encoding fails.
func createPrivateKeyPEM(privateKey *rsa.PrivateKey) (bytes.Buffer, error) {
	var privateKeyPEM bytes.Buffer
	err := pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	return privateKeyPEM, err
}

// createAndSave is the entry point of the program. It creates an X.509 certificate,
// generates the corresponding private key, and saves them to files
// cert.pem and privateKey.pem.
// Parameters:
//   - organization: the name of the organization for which the certificate is created.
//   - country: the country where the organization is registered.
//   - pathCertPEM: path where to save cert pem file.
//   - pathPrivateKeyPEM: path where to save private key pem file.
//
// Returns an error if operation fails.
func createAndSave(organization, country, pathCertPEM, pathPrivateKeyPEM string) error {
	privateKey, certBytes, err := createX509Certificate(organization, country)
	if err != nil {
		return err
	}

	certPEM, err := createCertPEM(certBytes)
	if err != nil {
		return err
	}

	privateKeyPEM, err := createPrivateKeyPEM(privateKey)
	if err != nil {
		return err
	}

	err = saveFile(pathCertPEM, certPEM.Bytes())
	if err != nil {
		return err
	}

	err = saveFile(pathPrivateKeyPEM, privateKeyPEM.Bytes())
	if err != nil {
		return err
	}
	return nil
}
