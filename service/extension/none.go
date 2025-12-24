package extension

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type None struct{}

func (p *None) Action(serviceId int32, action string) error {
	slog.Info("none action", "service id", serviceId, "action", action)
	return nil
}

func (p *None) ClientActions(serviceId int32) ([]string, error) {
	return []string{}, nil
}

func (p *None) AdminActions(serviceId int32) ([]string, error) {
	return []string{"suspend", "unsuspend", "terminate", "create"}, nil
}

func (p *None) Route(r chi.Router) error {
	return nil
}

func (p *None) ClientPage(w http.ResponseWriter, r *http.Request, serviceId int32) error {
	w.WriteHeader(http.StatusOK)
	return nil
}

func (p *None) AdminPage(w http.ResponseWriter, r *http.Request, serviceId int32) error {
	w.WriteHeader(http.StatusOK)
	return nil
}

func (p *None) Init() error {
	return nil
}

func (p *None) ProductSettings(inputs map[string]string) ([]ProductSetting, error) {
	return []ProductSetting{}, nil
}

func (p *None) ServerSettings() []ServerSettings {
	return []ServerSettings{}
}

func init() {
	registerExtension("None", &None{})
}
