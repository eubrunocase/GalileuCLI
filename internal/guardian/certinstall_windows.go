//go:build windows

package guardian

import (
	"crypto/sha1"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
)

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

func isAdmin() bool {
	cmd := exec.Command("net", "session")
	return cmd.Run() == nil
}

func certExistsInStore(certPath string) bool {
	thumbprint, err := getCertThumbprint(certPath)
	if err != nil {
		return false
	}

	cmd := exec.Command("certutil", "-store", "Root")
	output, _ := cmd.CombinedOutput()
	outputStr := string(output)
	for i := 0; i <= len(outputStr)-len(thumbprint); i++ {
		if outputStr[i:i+len(thumbprint)] == thumbprint {
			return true
		}
	}
	return false
}

// InstallCertificateIfNeeded no Windows instala automaticamente o certificado CA
// no repositório de certificados do sistema. Requer privilégios de Administrador.
func InstallCertificateIfNeeded(certPath string) error {
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return fmt.Errorf("certificado nao encontrado: %s", certPath)
	}

	if certExistsInStore(certPath) {
		fmt.Println("[GALILEU] Certificado CA ja esta instalado no repositorio.")
		return nil
	}

	if !isAdmin() {
		return fmt.Errorf("privilegios de administrador necessarios para instalar o certificado")
	}

	cmd := exec.Command("certutil", "-addstore", "-f", "Root", certPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha ao instalar certificado: %v, output: %s", err, string(output))
	}

	fmt.Println("[GALILEU] Certificado Root CA instalado com sucesso.")
	return nil
}
