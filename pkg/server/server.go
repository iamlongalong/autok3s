package server

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"os"
	"strings"

	"github.com/cnrancher/autok3s/pkg/server/proxy"
	"github.com/cnrancher/autok3s/pkg/server/ui"

	"github.com/gorilla/mux"
	responsewriter "github.com/rancher/apiserver/pkg/middleware"
	"github.com/rancher/apiserver/pkg/server"
	"github.com/rancher/apiserver/pkg/store/apiroot"
	"github.com/rancher/apiserver/pkg/types"

	// pprof
	"net/http/pprof"
)

// Start starts daemon.
func Start() http.Handler {
	s := server.DefaultAPIServer()
	initMutual(s.Schemas)
	initProvider(s.Schemas)
	initCluster(s.Schemas)
	initCredential(s.Schemas)
	initKubeconfig(s.Schemas)
	initLogs(s.Schemas)
	initTemplates(s.Schemas)
	initExplorer(s.Schemas)
	initSettings(s.Schemas)
	initPackage(s.Schemas)
	initSSHKey(s.Schemas)

	apiroot.Register(s.Schemas, []string{"v1"})
	router := mux.NewRouter()
	router.UseEncodedPath()
	router.StrictSlash(true)

	middleware := responsewriter.Chain{
		responsewriter.Gzip,
		responsewriter.FrameOptions,
		responsewriter.CacheMiddleware("json", "js", "css", "svg", "png", "woff", "woff2"),
		ui.ServeNotFound,
		ui.ServeJavascript,
		AuthMiddleware,
	}
	router.PathPrefix("/ui/").Handler(middleware.Handler(http.StripPrefix("/ui/", ui.Serve())))

	router.Path("/").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		http.Redirect(rw, req, "/ui/", http.StatusFound)
	})

	// profiling handlers for pprof under /debug/pprof.
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)

	// Manually add support for paths linked to by index page at /debug/pprof/
	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handle("/debug/pprof/block", pprof.Handler("block"))
	router.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))

	router.PathPrefix("/proxy/explorer/{name}").Handler(proxy.NewExplorerProxy())
	router.PathPrefix("/meta/proxy").Handler(proxy.NewProxy("/proxy/"))
	router.PathPrefix("/k8s/proxy").Handler(proxy.NewK8sProxy())
	router.Path("/{prefix}/{type}").Queries("action", "{action}").Handler(s)
	router.Path("/{prefix}/{type}").Handler(s)
	router.Path("/{prefix}/{type}/{name}").Queries("link", "{link}").Handler(s)
	router.Path("/{prefix}/{type}/{name}").Queries("action", "{action}").Handler(s)
	router.Path("/{prefix}/{type}/{name}").Handler(s)

	router.NotFoundHandler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		s.Handle(&types.APIRequest{
			Request:   r,
			Response:  rw,
			Type:      "apiRoot",
			URLPrefix: "v1",
		})
	})

	return router
}

func AuthMiddleware(next http.Handler) http.Handler {
	auths := map[string]string{}

	envs := os.Environ()
	for _, envStr := range envs {
		parts := strings.SplitN(envStr, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]
		if strings.HasPrefix(key, "AUTH_") {
			auths[strings.TrimPrefix(key, "AUTH_")] = value
		}
	}

	authToggle := os.Getenv("AUTOK3S_AUTH_TOGGLE")

	// use open_the_door md5 sum to aviod simple hack
	checksum := md5.New().Sum([]byte("open_the_door"))

	shouldSkipAuth := authToggle == string(checksum)

	magicSalt := "openit_"

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		if !shouldSkipAuth {
			uname, passwd, ok := r.BasicAuth()

			md5sum := md5.New()
			md5sum.Write([]byte(magicSalt + passwd))

			tarpasswd := hex.EncodeToString(md5sum.Sum(nil))

			pass := auths[uname]
			if !ok || pass != tarpasswd {
				// logrus.Warnf("auth fail, username: %s, pass: %s, tarPass: %s, shouldPass: %s", uname, passwd, tarpasswd, pass)
				rw.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(rw, r)
	})
}
