package extension

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

type None struct{}

func (p *None) Action(serviceId int32, action string) error {
	return nil
}

func (p *None) ClientActions(serviceId int32) ([]string, error) {
	return []string{}, nil
}

func (p *None) AdminActions(serviceId int32) ([]string, error) {
	return []string{}, nil
}

func (p *None) Route(r chi.Router) error {
	return nil
}

func (p *None) ClientPage(w http.ResponseWriter, serviceId int32) error {
	w.WriteHeader(http.StatusOK)
	return nil
}

func (p *None) AdminPage(w http.ResponseWriter, serviceId int32) error {
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
