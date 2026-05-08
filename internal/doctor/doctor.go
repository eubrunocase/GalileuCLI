package doctor

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
)

type DiagnosticResult struct {
	CertificateInstalled bool
	PortAvailable        bool
	EnvPortConfigured    bool
	PortNumber           int
	Errors               []string
}

func Diagnose() (*DiagnosticResult, error) {
	result := &DiagnosticResult{
		PortNumber: 9000,
	}
	if portEnv := os.Getenv("GALILEU_PORT"); portEnv != "" {
		if port, err := fmt.Sscanf(portEnv, "%d", &result.PortNumber); port == 1 && err == nil {
			result.EnvPortConfigured = true
		}
	}
	if err := checkCertificate(result); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}
	if err := checkPort(result.PortNumber, result); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}
	return result, nil
}
func checkCertificate(result *DiagnosticResult) error {
	certPath := filepath.Join("galileu-ca.pem")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return fmt.Errorf("certificado CA nao encontrado: %s", certPath)
	}
	osSpecificCheck := getOSCertificateCheck()
	if osSpecificCheck != nil && !osSpecificCheck() {
		return fmt.Errorf("certificado CA nao instalado no repositorio do sistema")
	}
	result.CertificateInstalled = true
	return nil
}
func checkPort(port int, result *DiagnosticResult) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("porta %d ja esta em uso", port)
	}
	ln.Close()
	result.PortAvailable = true
	return nil
}
func getOSCertificateCheck() func() bool {
	switch {
	case isLinux():
		return checkLinuxCertificate
	case isDarwin():
		return checkDarwinCertificate
	case isWindows():
		return checkWindowsCertificate
	}
	return nil
}
func isLinux() bool {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return false
	}
	return contains(string(data), "linux") || contains(string(data), "ID=")
}
func isDarwin() bool {
	cmd := exec.Command("uname")
	output, _ := cmd.Output()
	return contains(string(output), "Darwin")
}
func isWindows() bool {
	cmd := exec.Command("cmd", "/c", "echo")
	return cmd.Run() == nil
}
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
func checkLinuxCertificate() bool {
	paths := []string{
		"/usr/local/share/ca-certificates/galileu.crt",
		"/usr/share/ca-certificates/galileu.crt",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
func checkDarwinCertificate() bool {
	cmd := exec.Command("security", "find-certificate", "-c", "Galileu Local CA")
	return cmd.Run() == nil
}
func checkWindowsCertificate() bool {
	cmd := exec.Command("certutil", "-store", "Root")
	output, _ := cmd.CombinedOutput()
	return contains(string(output), "Galileu Local CA")
}
