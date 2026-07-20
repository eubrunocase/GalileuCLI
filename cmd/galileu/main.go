package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"Galileu/internal/ca"
	"Galileu/internal/doctor"
	"Galileu/internal/guardian"
	"Galileu/internal/tui"
)

const version = "2.0.0"

func main() {
	args := os.Args[1:]

	dryRun := containsArg(args, "--dry-run")
	useTUI := containsArg(args, "--tui")

	configPath, args := extractFlagValue(os.Args[1:], "--config")
	// Strip known flags before sub-command parsing.
	filteredArgs := filterArgs(args, "--dry-run", "--tui")

	if len(filteredArgs) == 0 {
		runProxy(dryRun, useTUI, configPath)
		return
	}


	switch filteredArgs[0] {
	case "doctor":
		runDoctor(configPath)
	case "version", "--version", "-v":
		runVersion()
	case "start":
		runProxy(dryRun, useTUI, configPath)
	case "-h", "--help", "help":
		printHelp()
	default:
		fmt.Printf("Comando desconhecido: %s\n", filteredArgs[0])
		printHelp()
		os.Exit(1)
	}
}

func filterArgs(args []string, exclude ...string) []string {
	excluded := make(map[string]bool, len(exclude))
	for _, e := range exclude {
		excluded[e] = true
	}
	result := make([]string, 0, len(args))
	for _, a := range args {
		if !excluded[a] {
			result = append(result, a)
		}
	}
	return result
}

func extractFlagValue(args []string, flag string) (string, []string) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			value := args[i+1]
			result := make([]string, 0, len(args)-2)
			result = append(result, args[:i]...)
			result = append(result, args[i+2:]...)
			return value, result
		}
	}
	return "", args
}

func containsArg(args []string, arg string) bool {
	for _, a := range args {
		if a == arg {
			return true
		}
	}
	return false
}

func printHelp() {
	fmt.Println(`Galileu - Proxy de Segurança para LLMs

Uso:
  galileu                     Iniciar o proxy (modo headless)
  galileu --tui               Iniciar o proxy com interface interactiva
  galileu --dry-run           Iniciar proxy em modo DRY-RUN (apenas detectar, não modificar)
  galileu doctor              Executar diagnóstico do sistema
  galileu version       	  Mostrar versão do binário
  galileu --config <path>     Usar arquivo de configuracao especifico

  Variaveis de ambiente:
  GALILEU_CONFIG              Caminho para o arquivo de configuracao
  GALILEU_PORT                Porta do proxy (usado pelo doctor)

Exemplos:
  galileu                         				Inicia o proxy na porta 9000
  galileu --config /etc/galileu/team.yml   		Config de equipe
  GALILEU_CONFIG=~/.config/galileu.yml galileu  Via variavel de ambiente
  galileu doctor                  				Verifica certificado, porta e variaveis
  galileu -h  									Mostra esta ajuda`)
}

func runVersion() {
	fmt.Printf("Galileu v%s\n", version)
}

func runDoctor(configPath string) {
	result, err := doctor.Diagnose(configPath)
	if err != nil {
		fmt.Printf("[ERRO] %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Diagnostico do Galileu ===")

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

	fmt.Printf("Config ativa: 	  ")
	switch result.ConfigSource {
	case "flag":
		fmt.Printf("[OK] %s (via --config)\n", result.ConfigPath)
	case "env":
		fmt.Printf("[OK] %s (via GALILEU_CONFIG)\n", result.ConfigPath)
	case "cwd":
		fmt.Printf("[OK] %s (CWD)\n", result.ConfigPath)
	case "builtin":
		fmt.Println("[OK] padroes built-in (nenhum arquivo)")
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

func runProxy(dryRun, withTUI bool, configFlag string) {
	certPath, keyPath := ca.ResolvePaths(ca.CertFile, ca.KeyFile)

	config, err := guardian.ResolveConfigPath(configFlag)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao resolver configuração: %v\n", err)
		os.Exit(1)
	}

	logConfigSource(config)

	certPEM, keyPEM, err := ca.EnsureCA(certPath, keyPath)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao garantir o certificado CA: %v\n", err)
		os.Exit(1)
	}

	if err := guardian.InstallCertificateIfNeeded(certPath); err != nil {
		fmt.Printf("[AVISO] %v\n", err)
	}

	port, patterns, err := guardian.LoadConfig(config.Path)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao carregar configuração: %v\n", err)
		os.Exit(1)
	}
	analyzer := guardian.NewAnalyzer(patterns)
	analyzer.DryRun = dryRun

	if withTUI {
		runProxyWithTUI(certPEM, keyPEM, analyzer, port, dryRun, config.Path)
	} else {
		runProxyPlain(certPEM, keyPEM, analyzer, port, dryRun)
	}
}

// runProxyWithTUI starts the proxy and drives the interactive TUI.
func runProxyWithTUI(certPEM, keyPEM []byte, analyzer *guardian.Analyzer, port int, dryRun bool, configPath string) {
	// Buffered channel: proxy goroutine never blocks on a slow TUI render.
	events := make(chan guardian.LogRequest, 128)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go guardian.GracefulListenWithCA(certPEM, keyPEM, analyzer, port, events)

	// Watch for OS signals and close the events channel to shut down the TUI.
	go func() {
		<-quit
		guardian.CloseGuardian()
		close(events)
	}()

	if err := tui.Start(port, dryRun, events, configPath); err != nil {
		fmt.Fprintf(os.Stderr, "[ERRO] TUI: %v\n", err)
	}

	guardian.CloseAuditLogger()
	fmt.Println("[GALILEU] Log de auditoria persistido com sucesso.")
}

// runProxyPlain is the headless mode — no TUI, plain text output.
func runProxyPlain(certPEM, keyPEM []byte, analyzer *guardian.Analyzer, port int, dryRun bool) {
	printBanner()

	if dryRun {
		fmt.Println("[GALILEU] Modo DRY-RUN ativo - apenas detectando, sem modificar payloads.")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go guardian.GracefulListenWithCA(certPEM, keyPEM, analyzer, port, nil)

	fmt.Println("[GALILEU] Proxy ativo na porta" + fmt.Sprintf(":%d", port) + ". Aguardando requisições...")
	fmt.Println("[GALILEU] Pressione Ctrl+C para encerrar e persistir o log de auditoria.")

	<-quit
	fmt.Println("\n[GALILEU] Encerrando...")
	guardian.CloseGuardian()
	guardian.CloseAuditLogger()
	fmt.Println("[GALILEU] Log de auditoria persistido com sucesso.")
}

func logConfigSource(config guardian.ConfigSource) {
	switch config.Source {
    case "flag":
        fmt.Printf("[GALILEU] config: usando --config=%s\n", config.Path)
    case "env":
        fmt.Printf("[GALILEU] config: usando GALILEU_CONFIG=%s\n", config.Path)
    case "cwd":
        fmt.Printf("[GALILEU] config: usando %s (CWD)\n", config.Path)
    case "builtin":
        fmt.Println("[GALILEU] config: nenhum arquivo encontrado, usando padroes built-in")
    }
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
