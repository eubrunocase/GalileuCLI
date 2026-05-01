package ca

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const (
	CertFile = "galileu-ca.pem"
	KeyFile  = "galileu-ca-key.pem"

	legacyCertFile = "ca.pem"
	legacyKeyFile  = "key.pem"
)

func EnsureCA(certPath, keyPath string) (certPEM, keyPEM []byte, err error) {
	if err := migrateLegacyCerts(certPath, keyPath); err != nil {
		return nil, nil, fmt.Errorf("falha ao migrar certificados antigos: %w", err)
	}

	certExists := fileExists(certPath)
	keyExists := fileExists(keyPath)

	if certExists && keyExists {
		certPEM, err = os.ReadFile(certPath)
		if err != nil {
			return nil, nil, fmt.Errorf("falha ao ler certificado: %w", err)
		}
		keyPEM, err = os.ReadFile(keyPath)
		if err != nil {
			return nil, nil, fmt.Errorf("falha ao ler chave privada: %w", err)
		}
		fmt.Println("[GALILEU] Certificado CA encontrado e carregado.")
		return certPEM, keyPEM, nil
	}

	if certExists || keyExists {
		fmt.Println("[GALILEU] Certificado CA incompleto. Gerando novo par de certificados...")
	} else {
		fmt.Println("[GALILEU] Certificado CA não encontrado. Gerando novo certificado...")
	}

	certPEM, keyPEM, err = GenerateCA()
	if err != nil {
		return nil, nil, err
	}

	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return nil, nil, fmt.Errorf("falha ao salvar certificado: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, nil, fmt.Errorf("falha ao salvar chave privada: %w", err)
	}

	fmt.Printf("[GALILEU] Certificado CA gerado com sucesso: %s\n", certPath)
	fmt.Printf("[GALILEU] Chave privada salva em: %s\n", keyPath)
	return certPEM, keyPEM, nil
}

func GenerateCA() (certPEM, keyPEM []byte, err error) {
	ca := &x509.Certificate{
		SerialNumber: bigSerial(),
		Subject: pkix.Name{
			Organization:  []string{"Galileu Security Proxy"},
			Country:       []string{""},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    "Galileu Local CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao gerar chave RSA: %w", err)
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao criar certificado: %w", err)
	}

	certBuf := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	keyBuf := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})

	return certBuf, keyBuf, nil
}

func ResolvePaths(certPath, keyPath string) (string, string) {
	if !filepath.IsAbs(certPath) {
		if wd, err := os.Getwd(); err == nil {
			certPath = filepath.Join(wd, certPath)
		}
	}
	if !filepath.IsAbs(keyPath) {
		if wd, err := os.Getwd(); err == nil {
			keyPath = filepath.Join(wd, keyPath)
		}
	}
	return certPath, keyPath
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func bigSerial() *big.Int {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	n, _ := rand.Int(rand.Reader, serialNumberLimit)
	return n
}

func migrateLegacyCerts(newCertPath, newKeyPath string) error {
	legacyCertExists := fileExists(legacyCertFile)
	legacyKeyExists := fileExists(legacyKeyFile)
	newCertExists := fileExists(newCertPath)
	newKeyExists := fileExists(newKeyPath)

	if !legacyCertExists || !legacyKeyExists {
		return nil
	}

	if newCertExists && newKeyExists {
		fmt.Println("[GALILEU] Novo certificado encontrado. Removendo certificados antigos (ca.pem, key.pem).")
		os.Remove(legacyCertFile)
		os.Remove(legacyKeyFile)
		return nil
	}

	fmt.Println("[GALILEU] Certificado antigo detectado (ca.pem, key.pem). Migrando para novo formato...")

	if err := os.Rename(legacyCertFile, newCertPath); err != nil {
		return fmt.Errorf("falha ao renomear %s para %s: %w", legacyCertFile, newCertPath, err)
	}
	if err := os.Rename(legacyKeyFile, newKeyPath); err != nil {
		return fmt.Errorf("falha ao renomear %s para %s: %w", legacyKeyFile, newKeyPath, err)
	}

	fmt.Printf("[GALILEU] Certificados migrados: %s -> %s\n", legacyCertFile, newCertPath)
	fmt.Printf("[GALILEU] Certificados migrados: %s -> %s\n", legacyKeyFile, newKeyPath)
	return nil
}
