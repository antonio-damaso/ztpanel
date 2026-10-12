// ztpanel is a self-hosted web UI for a ZeroTier network controller:
// a single Go binary serving a JSON API backed by the controller's
// local socket, plus the embedded Vue frontend.
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

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

	if err := bootstrapFirstAdmin(users); err != nil {
		log.Fatalf("could not bootstrap first admin user: %v", err)
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
	mux.Handle("/", webServer)

	log.Printf("ztpanel listening on %s (controller: %s)", cfg.ListenAddr, cfg.ZTControllerURL)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatal(err)
	}
}

// bootstrapFirstAdmin creates a single "admin" user with a freshly
// generated random password the first time ztpanel runs against an
// empty user store, and prints it once. There is no hardcoded default
// password anywhere in this codebase.
func bootstrapFirstAdmin(users *authsvc.UserStore) error {
	count, err := users.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	password, err := authsvc.GenerateRandomPassword()
	if err != nil {
		return err
	}
	if err := users.Create("admin", password, true); err != nil {
		return err
	}

	banner := strings.Repeat("=", 60)
	fmt.Println(banner)
	fmt.Println("ztpanel: first run detected, created an initial admin account")
	fmt.Println("  username: admin")
	fmt.Printf("  password: %s\n", password)
	fmt.Println("You will be asked to set a new password on first login.")
	fmt.Println(banner)
	return nil
}
