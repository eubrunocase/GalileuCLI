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
                                                                                                                                             
                       (     (    (                 
 (        (      )\ )  )\ ) )\ )              
 )\ )     )\    (()/( (()/((()/(  (       (   
(()/(  ((((_)(   /(_)) /(_))/(_)) )\      )\  
 /(_))_ )\ _ )\ (_))  (_)) (_))  ((_)  _ ((_) 
(_)) __|(_)_\(_)| |   |_ _|| |   | __|| | | | 
  | (_ | / _ \  | |__  | | | |__ | _| | |_| | 
   \___|/_/ \_\ |____||___||____||___| \___/  
                                                   
    `)

	certPath, keyPath := ca.ResolvePaths(ca.CertFile, ca.KeyFile)

	certPEM, keyPEM, err := ca.EnsureCA(certPath, keyPath)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao garantir o certificado CA: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[GALILEU] Instalando certificado CA no Keychain...")
	if err := ca.InstallCert(certPath); err != nil {
		fmt.Printf("[AVISO] Nao foi possivel instalar o certificado automaticamente: %v\n", err)
		fmt.Println("[DICA] Execute: sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain " + certPath)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go guardian.GracefulListenWithCA(certPEM, keyPEM)

	fmt.Println("[GALILEU] Proxy ativo na porta 9000.")
	fmt.Println("[GALILEU] Pressione Ctrl+C para encerrar e persistir o log de auditoria.")

	<-quit
	fmt.Println("\n[GALILEU] Encerrando...")
	guardian.CloseGuardian()
	guardian.CloseAuditLogger()
	fmt.Println("[GALILEU] Log de auditoria persistido com sucesso.")
}
