package api

import (
	"awesomeProject/client"
	service "awesomeProject/external"
	"encoding/json"
	httpSwagger "github.com/swaggo/http-swagger"
	"net/http"
)

type API struct {
	Client client.Client
}

func NewAPI(client *client.Client) *API {
	return &API{
		Client: *client,
	}
}

// ProcessBatchHandler godoc
// @Summary Process a batch of items
// @Description Process a batch of items through the service
// @Tags batch
// @Accept  json
// @Produce  json
// @Param   batch  body      []Item  true  "Batch to process"
// @Success 200
// @Router /process-batch [post]
func (api *API) ProcessBatchHandler(w http.ResponseWriter, r *http.Request) {
	// For simplicity, assuming the batch is sent as a JSON array in the request body
	var batch []service.Item
	err := json.NewDecoder(r.Body).Decode(&batch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	api.Client.EnqueueItems(batch)

	w.WriteHeader(http.StatusOK)
}

// SetupRoutes sets up the API routes
func (api *API) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/process-batch", api.ProcessBatchHandler)
	mux.HandleFunc("/swagger/*", httpSwagger.WrapHandler)
}
