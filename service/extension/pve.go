package extension

import (
	"billing3/database"
	"billing3/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type PVE struct {
	httpClient http.Client
}

type pveResp[T any] struct {
	Data T `json:"data"`
}

func (p *PVE) getApiBaseURL(server *database.Server) string {
	u := url.URL{
		Scheme: "https",
		Host:   server.Settings["address"] + ":" + server.Settings["port"],
		Path:   "api2/json",
	}
	return u.String()
}

// pveAuth returns CSRFPreventionToken and ticket
func (p *PVE) pveAuth(base string, username string, password string) (string, string, error) {

}

func (p *PVE) Action(serviceId int32, action string) error {
	switch action {
	case "poweroff":
		return nil
	case "reboot":
		return nil
	case "suspend":
		return nil
	case "unsuspend":
		return nil
	case "terminate":
		return nil
	case "create":
		s, err := database.Q.FindServiceById(context.Background(), serviceId)
		if err != nil {
			return fmt.Errorf("pve: %w", err)
		}

		cpu := s.Settings["cpu"]
		disk := s.Settings["disk"]
		memory := s.Settings["memory"]
		servers := s.Settings["servers"]

		serverIds := make([]int, 0)
		for _, s := range strings.Split(servers, ",") {
			i, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("pve: invalid servers: %s", servers)
			}
			serverIds = append(serverIds, i)
		}

		if len(serverIds) == 0 {
			return fmt.Errorf("pve: no servers available")
		}

		// choose a random pve server
		serverId := serverIds[utils.Randint(0, len(serverIds)-1)]

		// pve server settings
		server, err := database.Q.FindServerById(context.Background(), int32(serverId))
		if err != nil {
			return fmt.Errorf("pve: invalid servers: %d %w", serverId, err)
		}

		address := server.Settings["address"]
		port := server.Settings["port"]
		username := server.Settings["username"]
		password := server.Settings["password"]
		ips := server.Settings["ips"]

		// save server id
		s.Settings["server"] = strconv.Itoa(serverId)

		err = database.Q.UpdateServiceSettings(context.Background(), database.UpdateServiceSettingsParams{
			ID:       int32(serviceId),
			Settings: s.Settings,
		})
		if err != nil {
			return fmt.Errorf("pve: %w", err)
		}

		return nil
	}

	return fmt.Errorf("invalid action \"%s\"", action)
}

func (p *PVE) ClientActions(serviceId int32) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PVE) AdminActions(serviceId int32) ([]string, error) {
	return []string{"poweroff", "reboot", "terminate", "suspend", "unsuspend", "create"}, nil
}

func (p *PVE) Route(r chi.Router) error {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello, world")
	})
	return nil
}

func (p *PVE) ClientPage(w http.ResponseWriter, serviceId int32) error {
	//TODO implement me
	panic("implement me")
}

func (p *PVE) AdminPage(w http.ResponseWriter, serviceId int32) error {
	io.WriteString(w, "<p>Memory: 123MB / 1024MB</p><p>Bandwidth: 1G / 1024G</p><p>Disk: 5G / 20G</p><p>CPU: 30%</p>")
	return nil
}

func (p *PVE) Init() error {
	p.httpClient = http.Client{
		Timeout: time.Second * 10,
	}
	return nil
}

func (p *PVE) ProductSettings(inputs map[string]string) ([]ProductSetting, error) {
	return []ProductSetting{
		{Name: "disk", DisplayName: "Disk (GB)", Type: "string", Regex: "^\\d+"},
		{Name: "memory", DisplayName: "Memory (MB)", Type: "string", Regex: "^\\d+"},
		{Name: "cpu", DisplayName: "CPU Cores", Type: "string", Regex: "^\\d+"},
		{Name: "servers", DisplayName: "Servers", Type: "servers"},
	}, nil
}

func (p *PVE) ServerSettings() []ServerSettings {
	return []ServerSettings{
		{Name: "address", DisplayName: "Address", Type: "string", Placeholder: "8.8.8.8", Regex: "^.+$"},
		{Name: "port", DisplayName: "Port", Type: "string", Placeholder: "8006", Regex: "^\\d+$"},
		{Name: "username", DisplayName: "Username", Type: "string", Placeholder: "root@pam", Regex: "^.+$"},
		{Name: "password", DisplayName: "Password", Type: "string", Regex: "^.+$"},
		{Name: "ips", DisplayName: "IP Addresses (one per line)", Type: "text", Placeholder: "10.2.3.100/24\n10.2.3.101/24\n10.2.3.102/24\n10.2.3.103/24"},
	}
}

func init() {
	registerExtension("PVE", &PVE{})
}
