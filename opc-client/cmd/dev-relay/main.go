// Developer-only real HTTP transport for browser integration tests. No native credentials.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Tencent/WeKnora/opc-client/internal/cloud"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	origin := flag.String("origin", cloud.DefaultOrigin, "selected cloud origin")
	frontend := flag.String("frontend-origin", "http://127.0.0.1:5174", "explicit development UI origin")
	file := flag.String("config", ".runtime.json", "runtime config output (private, ignored)")
	httpDev := flag.Bool("allow-http", false, "allow HTTP on private/loopback addresses")
	flag.Parse()
	u, err := cloud.NormalizeOrigin(*origin, *httpDev)
	if err != nil {
		panic(err)
	}
	r, err := cloud.NewRelay(u, *frontend)
	if err != nil {
		panic(err)
	}
	defer r.Close()
	data, _ := json.Marshal(cloud.Config{PublicOrigin: u, ProfileID: cloud.ProfileID(u), TransportBase: r.URL, RelayCapability: r.Capability, RemoteDesktop: true, Version: "development"})
	if err = os.WriteFile(*file, data, 0600); err != nil {
		panic(err)
	}
	defer os.Remove(*file)
	fmt.Println("OPC development relay ready; capability is kept in the runtime file, not logs")
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	<-done
}
