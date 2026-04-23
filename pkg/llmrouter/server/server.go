package server

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/lx1036/gateway/pkg/llmrouter/router"
	"k8s.io/klog/v2"
	"net/http"
)

type Server struct {

}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Run(ctx context.Context){
	r := router.NewRouter()


	// start gin server
	engine := gin.New()
	engine.Use(gin.Recovery())

	engine.Any("/*path", r.HandlerFunc())

	server := &http.Server{
		Addr:    ":8000",
		Handler: engine,
	}

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			klog.Errorf("listen: %s\n", err)
		}
	}()


	<-ctx.Done()
}





