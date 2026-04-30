package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go guardian.GracefulListen()

	fmt.Println("[GALILEU] Proxy ativo na porta 9000.")
	fmt.Println("[GALILEU] Pressione Ctrl+C para encerrar e persistir o log de auditoria.")

	<-quit
	fmt.Println("\n[GALILEU] Encerrando...")
	guardian.CloseGuardian()
	guardian.CloseAuditLogger()
	fmt.Println("[GALILEU] Log de auditoria persistido com sucesso.")
}

