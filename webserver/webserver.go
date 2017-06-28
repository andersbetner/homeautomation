package webserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
)

// Webserver creates a webserver that gracefully shuts down on ctrl-C
func Webserver(name string, address string, handler http.Handler) {
	srv := &http.Server{Addr: address, Handler: handler}
	go func() {
		done := make(chan os.Signal)
		signal.Notify(done, os.Interrupt)
		<-done
		fmt.Println("Shutting down " + name)
		srv.Shutdown(context.Background())
	}()
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		fmt.Println(err)
		panic("Can't start " + name)
	}
}
