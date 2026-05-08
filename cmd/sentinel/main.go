package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"Galileu/internal/ca"
	"Galileu/internal/doctor"
	"Galileu/internal/guardian"
)

const version = "1.0.0"

func main() {
	args := os.Args[1:]

	dryRun := containsArg(args, "--dry-run")

	filteredArgs := filterArgs(args, "--dry-run")

	if len(filteredArgs) == 0 {
		runProxy(dryRun)
		return
	}

	switch filteredArgs[0] {
	case "doctor":
		runDoctor()
	case "version", "--version", "-v":
		runVersion()
	case "start":
		runProxy(dryRun)
	case "-h", "--help", "help":
		printHelp()
	default:
		fmt.Printf("Comando desconhecido: %s\n", filteredArgs[0])
		printHelp()
		os.Exit(1)
	}
}

func filterArgs(args []string, exclude string) []string {
	result := make([]string, 0)
	for _, a := range args {
		if a != exclude {
			result = append(result, a)
		}
	}
	return result
}

func containsArg(args []string, arg string) bool {
	for _, a := range args {
		if a == arg {
			return true
		}
	}
	return false
}

func startsWithDash(s string) bool {
	return len(s) > 0 && s[0] == '-'
}

func printHelp() {
	fmt.Println(`Galileu - Proxy de Segurança para LLMs

Uso:
  galileu               Iniciar o proxy
  galileu --dry-run     Iniciar proxy em modo DRY-RUN (apenas detectar, não modificar)
  galileu doctor        Executar diagnóstico do sistema
  galileu version       Mostrar versão do binário

Exemplos:
  galileu               Inicia o proxy na porta 9000
  galileu --dry-run     Inicia proxy em modo teste (mostra o que seria redatado)
  galileu doctor        Verifica certificado, porta e variáveis
  galileu -h            Mostra esta ajuda`)
}

func runVersion() {
	fmt.Printf("Galileu v%s\n", version)
}

func runDoctor() {
	result, err := doctor.Diagnose()
	if err != nil {
		fmt.Printf("[ERRO] %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Diagnostico do Galileu ===\n")

	fmt.Printf("Certificado CA:      ")
	if result.CertificateInstalled {
		fmt.Println("[OK] Instalado")
	} else {
		fmt.Println("[FALHA] Nao instalado")
	}

	fmt.Printf("Porta configurada:    ")
	if result.EnvPortConfigured {
		fmt.Printf("[OK] %d (via GALILEU_PORT)\n", result.PortNumber)
	} else {
		fmt.Printf("[OK] %d (padrao)\n", result.PortNumber)
	}

	fmt.Printf("Porta disponivel:    ")
	if result.PortAvailable {
		fmt.Println("[OK] Livre")
	} else {
		fmt.Println("[FALHA] Ja em uso")
	}

	fmt.Println("")

	if len(result.Errors) > 0 {
		fmt.Println("Problemas encontrados:")
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("Tudo OK!")
}

func runProxy(dryRun bool) {
	printBanner()

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
	analyzer.DryRun = dryRun

	if dryRun {
		fmt.Println("[GALILEU] Modo DRY-RUN ativo - apenas detectando, sem modificar payloads.")
	}

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

func printBanner() {
	fmt.Println(`
 ██████╗  █████╗ ██╗     ██╗██╗     ███████╗██╗   ██╗
██╔════╝ ██╔══██╗██║     ██║██║     ██╔════╝██║   ██║
██║  ███╗███████║██║     ██║██║     █████╗  ██║   ██║
██║   ██║██╔══██║██║     ██║██║     ██╔══╝  ██║   ██║
╚██████╔╝██║  ██║███████╗██║███████╗███████╗╚██████╔╝
 ╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝╚══════╝╚══════╝ ╚═════╝ 
                                                      `)
}
