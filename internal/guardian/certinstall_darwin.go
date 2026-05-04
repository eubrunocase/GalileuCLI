//go:build darwin

package guardian

import "fmt"

// InstallCertificateIfNeeded no macOS informa o utilizador que a instalação
// do certificado CA deve ser feita manualmente no Keychain Access.
// Não requer privilégios de administrador.
func InstallCertificateIfNeeded(certPath string) error {
	fmt.Println("[Galileu] macOS detectado.")
	fmt.Printf("[Galileu] Certifique-se de que '%s' está importado no Keychain Access com confiança 'Always Trust'.\n", certPath)
	fmt.Println("[Galileu] Consulte o README para instruções detalhadas.")
	return nil
}
