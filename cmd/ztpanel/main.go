// ztpanel is a self-hosted web UI for a ZeroTier network controller:
// a single Go binary serving a JSON API backed by the controller's
// local socket, plus the embedded Vue frontend.
package main

import (
	"log"
	"net/http"

	"github.com/antonio-damaso/ztpanel/internal/api"
	"github.com/antonio-damaso/ztpanel/internal/apitoken"
	"github.com/antonio-damaso/ztpanel/internal/auditsvc"
	"github.com/antonio-damaso/ztpanel/internal/authsvc"
	"github.com/antonio-damaso/ztpanel/internal/config"
	"github.com/antonio-damaso/ztpanel/internal/orgsvc"
	"github.com/antonio-damaso/ztpanel/internal/ztapi"
	"github.com/antonio-damaso/ztpanel/web"
)

func main() {
	cfg := config.Load()

	if err := config.EnsureDataDir(cfg.DataDir); err != nil {
		log.Fatalf("could not prepare data dir %s: %v", cfg.DataDir, err)
	}

	zt, err := ztapi.New(cfg.ZTControllerURL, cfg.ZTAuthTokenPath)
	if err != nil {
		log.Fatalf("could not connect to zerotier controller: %v", err)
	}

	users, err := authsvc.NewUserStore(cfg.UsersFile())
	if err != nil {
		log.Fatalf("could not open user store: %v", err)
	}

	sessions, err := authsvc.NewSessionSigner(cfg.SessionKeyFile())
	if err != nil {
		log.Fatalf("could not initialise sessions: %v", err)
	}

	orgs, err := orgsvc.NewStore(cfg.OrganizationsFile())
	if err != nil {
		log.Fatalf("could not open organizations store: %v", err)
	}

	audit, err := auditsvc.NewLog(cfg.ActivityLogFile())
	if err != nil {
		log.Fatalf("could not open activity log: %v", err)
	}

	tokens, err := apitoken.NewStore(cfg.TokensFile())
	if err != nil {
		log.Fatalf("could not open API token store: %v", err)
	}

	apiServer := api.NewServer(zt, users, sessions, orgs, audit, tokens)
	webServer := web.Handler()

	mux := http.NewServeMux()
	mux.Handle("/api/", apiServer)
	mux.Handle("/mcp", apiServer)
	mux.Handle("/healthz", apiServer)
	mux.Handle("/", webServer)

	log.Printf("ztpanel listening on %s (controller: %s)", cfg.ListenAddr, cfg.ZTControllerURL)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatal(err)
	}
}
