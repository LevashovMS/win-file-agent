package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Server interface {
	Run(ctx context.Context) error
	Stop()
}

type server struct {
	ctx    context.Context // общий контекст
	cf     context.CancelFunc
	srv    *http.Server // HTTP‑сервер
	port   int
	router *router
}

func New(args ...ArgsHandler) Server {
	var s = &server{router: newRouter()}
	for _, it := range args {
		it(s)
	}

	return s
}

func (c *server) Run(ctx context.Context) (err error) {
	if err = c.verification(); err != nil {
		return err
	}

	c.ctx, c.cf = context.WithCancel(ctx)
	var addrPort = fmt.Sprintf(":%d", c.port)
	// 1) Запускаем HTTP‑сервер
	c.srv = &http.Server{
		Addr:              addrPort,
		Handler:           c.router.mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second, // Maximum time to read request headers

	}
	go func() {
		err = c.srv.ListenAndServe()
	}()
	if err != nil {
		return err
	}

	log.Printf("Запуск сервера с адресом: %s\n", addrPort)
	return nil
}

func (c *server) Stop() {
	// 2) Завершаем HTTP‑сервер с таймаутом
	var ctx, cf = context.WithTimeout(context.Background(), 15*time.Second)
	defer cf()
	if err := c.srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
}
