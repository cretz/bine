package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cretz/bine/tor"
	"github.com/cretz/bine/torutil"
)

var verbose bool

func main() {
	flag.BoolVar(&verbose, "verbose", false, "Whether to have verbose logging")
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("Expecting single domain arg")
	} else if err := run(flag.Arg(0)); err != nil {
		log.Fatal(err)
	}
}

func run(domain string) error {
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	// Make sure mkcert is available
	if _, err := exec.LookPath("mkcert"); err != nil {
		return fmt.Errorf("Unable to find mkcert on PATH: %v", err)
	}
	// Listen until enter pressed
	srv, err := start(ctx, domain, ":80")
	if err != nil {
		return err
	}
	defer srv.Close()
	fmt.Printf("Listening on all IPs on port 80, so http://%v will use second onion as alt-svc\n", domain)
	fmt.Printf("Listening on onion http://%v.onion that will use second onion as alt-svc\n", srv.onion1.ID)
	fmt.Printf("Created secure second onion at https://%v.onion\n", srv.onion2.ID)
	fmt.Println("Press enter to exit")
	// Wait for key asynchronously
	go func() {
		fmt.Scanln()
		cancelFn()
	}()
	select {
	case err := <-srv.Err():
		return err
	case <-ctx.Done():
		return nil
	}
}

type server struct {
	exitAddrs    map[string]bool
	t            *tor.Tor
	onion1       *tor.OnionService
	onion2       *tor.OnionService
	httpSrv      *http.Server
	httpSrvErrCh chan error
}

func start(ctx context.Context, domain string, httpAddr string) (srv *server, err error) {
	srv = &server{}
	// // Get all exit addrs
	if srv.exitAddrs, err = getExitAddresses(); err != nil {
		return nil, err
	}
	// Start tor
	startConf := &tor.StartConf{DataDir: "tor-data"}
	if verbose {
		startConf.DebugWriter = os.Stdout
	} else {
		startConf.ExtraArgs = []string{"--quiet"}
	}
	if srv.t, err = tor.Start(ctx, startConf); err != nil {
		return nil, err
	}
	// Henceforth, any err needs to call close
	// Start Onion 1
	if srv.onion1, err = srv.t.Listen(ctx, &tor.ListenConf{RemotePorts: []int{80}, Version3: true}); err != nil {
		srv.Close()
		return nil, err
	}
	// Start Onion 2
	if srv.onion2, err = srv.t.Listen(ctx, &tor.ListenConf{RemotePorts: []int{443}, Version3: true}); err != nil {
		srv.Close()
		return nil, err
	}
	// Call mkcert for both onions
	cmd := exec.CommandContext(ctx, "mkcert", domain, srv.onion1.ID+".onion", srv.onion2.ID+".onion")
	cmd.Dir = "tor-data"
	output, err := cmd.CombinedOutput()
	if verbose {
		fmt.Printf("Output from mkcert:\n%v\n", string(output))
	}
	if err != nil {
		srv.Close()
		return nil, fmt.Errorf("Failed running mkcert: %v", err)
	}
	cert := filepath.Join("tor-data", domain+"+2.pem")
	key := filepath.Join("tor-data", domain+"+2-key.pem")
	// Listen on the onions
	srv.httpSrvErrCh = make(chan error, 3)
	go func(errCh chan error) {
		errCh <- http.Serve(srv.onion1, srv.NewHandler(srv.onion1.ID+".onion", srv.onion2.ID+".onion:443"))
	}(srv.httpSrvErrCh)
	go func(errCh chan error) {
		errCh <- http.ServeTLS(srv.onion2, srv.NewHandler(srv.onion2.ID+".onion", ""), cert, key)
	}(srv.httpSrvErrCh)
	// Start HTTP server
	srv.httpSrv = &http.Server{Addr: httpAddr, Handler: srv.NewHandler(httpAddr, srv.onion2.ID+".onion:443")}
	go func(httpSrv *http.Server, errCh chan error) { errCh <- httpSrv.ListenAndServe() }(srv.httpSrv, srv.httpSrvErrCh)
	return
}

func (s *server) Err() <-chan error { return s.httpSrvErrCh }

func (s *server) NewHandler(siteAddr string, altSvc string) http.Handler {
	hitCount := 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitCount++
		fmt.Printf("Accessed %v (%v)\n", siteAddr, hitCount)
		// Set an alt-svc header
		if altSvc != "" {
			w.Header().Add("Alt-Svc", "h2=\""+altSvc+"\"; ma=600")
		}
		// Respond
		remoteAddr, _, _ := torutil.PartitionString(r.RemoteAddr, ':')
		exit := ""
		if !s.exitAddrs[remoteAddr] {
			exit = " NOT"
		}
		fmt.Fprintf(w, "Server-side site addr: %v\n", siteAddr)
		fmt.Fprintf(w, "You accessed %v on %v from %v which is%v an exit node\n",
			r.URL.Path, r.Host, r.RemoteAddr, exit)
		fmt.Fprintf(w, "--------------- Headers ---------------\n")
		for h, vals := range r.Header {
			for _, val := range vals {
				fmt.Fprintf(w, "%v: %v\n", h, val)
			}
		}
	})
}

func (s *server) Close() {
	if s.httpSrv != nil {
		s.httpSrv.Close()
	}
	if s.onion1 != nil {
		s.onion1.Close()
	}
	if s.onion2 != nil {
		s.onion2.Close()
	}
	if s.t != nil {
		s.t.Close()
	}
}

func getExitAddresses() (map[string]bool, error) {
	resp, err := http.Get("https://check.torproject.org/exit-addresses")
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	ret := map[string]bool{}
	for _, line := range strings.Split(string(body), "\n") {
		pieces := strings.Split(strings.TrimSpace(line), " ")
		if len(pieces) >= 2 && pieces[0] == "ExitAddress" {
			fmt.Printf("Found exit: '%v'\n", pieces[1])
			ret[pieces[1]] = true
		}
	}
	return ret, nil
}
