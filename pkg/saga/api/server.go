package api

import (
	"github.com/kopaygorodsky/brigadier/pkg/log"
	api "github.com/kopaygorodsky/brigadier/pkg/saga/api/handlers/status"
	"net/http"
)

func StartServer(addr string, logger log.Logger, statusHandler *api.StatusHandler) error {
	m := http.NewServeMux()

	m.HandleFunc("/sagas", statusHandler.GetFilteredBy)
	m.HandleFunc("/sagas/", statusHandler.GetStatus)

	go func() {
		logger.Logf(log.InfoLevel, "Started saga api server on `%s`", addr)
		logger.Log(log.PanicLevel, http.ListenAndServe(addr, m))
	}()

	return nil
}
