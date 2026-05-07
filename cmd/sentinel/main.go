package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"Galileu/internal/ca"
	"Galileu/internal/guardian"
)

func main() {
	fmt.Println(`                                                                   
  ▒███▒   ██   █      █████  █      ██████ █    █
 ░█▒ ░█   ██   █        █    █      █      █    █
 █▒      ▒██▒  █        █    █      █      █    █
 █       ▓▒▒▓  █        █    █      █      █    █
 █   ██  █░░█  █        █    █      ██████ █    █
 █    █  █  █  █        █    █      █      █    █
 █▒   █ ▒████▒ █        █    █      █      █    █
 ▒█░ ░█ ▓▒  ▒▓ █        █    █      █      █▒  ▒█
  ▒███▒ █░  ░█ ██████ █████  ██████ ██████  ████ `)

	certPath, keyPath := ca.ResolvePaths(ca.CertFile, ca.KeyFile)

	certPEM, keyPEM, err := ca.EnsureCA(certPath, keyPath)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao garantir o certificado CA: %v\n", err)
		os.Exit(1)
	}

	if err := guardian.InstallCertificateIfNeeded(certPath); err != nil {
		fmt.Printf("[AVISO] %v\n", err)
	}

	port, patterns, err := guardian.LoadConfig("galileu.yml")
	if err != nil {
		fmt.Printf("[ERRO] Falha ao carregar configuração: %v\n", err)
		os.Exit(1)
	}
	analyzer := guardian.NewAnalyzer(patterns)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go guardian.GracefulListenWithCA(certPEM, keyPEM, analyzer, port)

	fmt.Println("[GALILEU] Proxy ativo na porta" + fmt.Sprintf(":%d", port) + ". Aguardando requisições...")
	fmt.Println("[GALILEU] Pressione Ctrl+C para encerrar e persistir o log de auditoria.")

	<-quit
	fmt.Println("\n[GALILEU] Encerrando...")
	guardian.CloseGuardian()
	guardian.CloseAuditLogger()
	fmt.Println("[GALILEU] Log de auditoria persistido com sucesso.")
}
