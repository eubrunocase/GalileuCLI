//go:build linux

package guardian

import "fmt"

// InstallCertificateIfNeeded no Linux informa o utilizador que a instalação
// do certificado CA deve ser feita manualmente no sistema de certificados da distribuição.
func InstallCertificateIfNeeded(certPath string) error {
	fmt.Println("[Galileu] Linux detectado.")
	fmt.Printf("[Galileu] Para instalar o certificado CA, execute:\n")
	fmt.Printf("  sudo cp %s /usr/local/share/ca-certificates/galileu.crt\n", certPath)
	fmt.Println("  sudo update-ca-certificates")
	return nil
}
