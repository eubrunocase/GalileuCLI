package ca

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func InstallCert(certPath string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("instalacao automatica suportada apenas no macOS")
	}

	if isCertTrusted(certPath) {
		fmt.Println("[GALILEU] Certificado CA ja esta instalado e confiado no Keychain.")
		return nil
	}

	fmt.Println("[GALILEU] Instalando certificado CA no Keychain do sistema...")

	cmd := exec.Command("sudo", "security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", "/Library/Keychains/System.keychain", certPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha ao instalar certificado: %v\n%s", err, string(output))
	}

	fmt.Println("[GALILEU] Certificado CA instalado com sucesso no Keychain do sistema.")
	return nil
}

func isCertTrusted(certPath string) bool {
	cmd := exec.Command("security", "find-certificate", "-c", "Galileu Local CA", "-p")
	output, err := cmd.CombinedOutput()
	if err != nil || len(output) == 0 {
		return false
	}

	readCmd := exec.Command("cat", certPath)
	readOutput, err := readCmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), strings.TrimSpace(string(readOutput)))
}
