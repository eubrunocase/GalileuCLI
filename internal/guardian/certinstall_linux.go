//go:build linux

package guardian

import (
	"crypto/sha1"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const linuxCertPath = "/usr/local/share/ca-certificates/galileu.crt"

func getCertThumbprint(certPath string) (string, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return "", err
	}

	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("arquivo de certificado invalido")
	}

	hash := sha1.Sum(block.Bytes)
	return fmt.Sprintf("%X", hash), nil
}

func isRoot() bool {
	return os.Geteuid() == 0
}

func certInstalledInSystem(certPath string) bool {
	thumbprint, err := getCertThumbprint(certPath)
	if err != nil {
		return false
	}

	data, err := os.ReadFile(linuxCertPath)
	if err != nil {
		return false
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return false
	}

	installedHash := sha1.Sum(block.Bytes)
	installedThumbprint := fmt.Sprintf("%X", installedHash)

	return thumbprint == installedThumbprint
}

func InstallCertificateIfNeeded(certPath string) error {
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return fmt.Errorf("certificado nao encontrado: %s", certPath)
	}

	if certInstalledInSystem(certPath) {
		fmt.Println("[GALILEU] Certificado CA ja esta instalado no repositorio.")
		return nil
	}

	if !isRoot() {
		fmt.Println("[Galileu] Linux detectado.")
		fmt.Printf("[Galileu] Para instalar o certificado CA, execute:\n")
		fmt.Printf("  sudo cp %s %s\n", certPath, linuxCertPath)
		fmt.Println("  sudo update-ca-certificates")
		return nil
	}

	certDir := filepath.Dir(linuxCertPath)
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar diretorio de certificados: %w", err)
	}

	if err := copyFile(certPath, linuxCertPath); err != nil {
		return fmt.Errorf("falha ao copiar certificado: %w", err)
	}

	cmd := exec.Command("update-ca-certificates")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha ao atualizar certificados do sistema: %v, output: %s", err, string(output))
	}

	fmt.Println("[GALILEU] Certificado Root CA instalado com sucesso.")
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
