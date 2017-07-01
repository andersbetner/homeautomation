package util

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"

	log "github.com/Sirupsen/logrus"
)

// Webserver creates a webserver that gracefully shuts down on ctrl-C
func Webserver(name string, address string, handler http.Handler) {
	srv := &http.Server{Addr: address, Handler: handler}
	go func() {
		done := make(chan os.Signal)
		signal.Notify(done, os.Interrupt)
		<-done
		log.Info("Shutting down ", name)
		srv.Close()
	}()
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		fmt.Println(err)
		panic("Can't start " + name)
	}
}
