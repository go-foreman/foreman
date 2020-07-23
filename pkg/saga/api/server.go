package api

import (
	api "github.com/kopaygorodsky/brigadier/pkg/saga/api/handlers/status"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
)

func StartServer(logger *log.Logger, statusHandler *api.StatusHandler) error {
	m := http.NewServeMux()

	m.HandleFunc("/sagas", statusHandler.GetFilteredBy)
	m.HandleFunc("/sagas/", statusHandler.GetStatus)

	address := viper.GetString("saga.api.server.address")

	if address == "" {
		address = ":8050"
	}

	go func() {
		logger.Infof("Started saga api server on `%s`", address)
		logger.Fatal(http.ListenAndServe(address, m))
	}()

	return nil
}
