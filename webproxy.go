package main

import (
    "net/http"
    "net/http/httputil"
    "net/url"
    "sync"
    "flag"
    "os"
    "time"
    "log"
)

const (
    usage = "webproxy [options]\n"
)

var (
    sockpath = ""
    help = false
    mu sync.Mutex
    count int
    listen string
    listentls string
    cert string
    key string
    colour bool
    documentRoot string
    sleepTime time.Duration = time.Second * 10
    logger *log.Logger
)

func init() {
    flag.BoolVar(&help, "h", false, "Print help")
    flag.StringVar(&listen, "l", "0.0.0.0:80", "Listen address for http (blank for none)")
    /*flag.StringVar(&listentls, "t", "0.0.0.0:443", "Listen address for https (blank for none)")
    flag.StringVar(&cert, "c", "cert.pem", "Path to cert.pem file")
    flag.StringVar(&key, "k", "key.pem", "Path to key.pem file") */
    flag.StringVar(&documentRoot, "r", "/var/www/html", "Document root to serve from")
    flag.Parse()

    if help {
        flag.PrintDefaults()
        os.Exit(1)
    }

    logger = log.New(os.Stderr, "", 0)
}

/*
 * The status response writer, to capture response status for logging.
 */

type statusResponseWriter struct {
    http.ResponseWriter
    statusCode int
}

func NewStatusResponseWriter(w http.ResponseWriter) *statusResponseWriter {
    return &statusResponseWriter{w, http.StatusOK}
}

func (sw statusResponseWriter) WriteHeader(code int) {
    sw.statusCode = code
    sw.ResponseWriter.WriteHeader(code)
}

/*
 * This is a logging wrapper for all of the http handlers, so we don't
 * have to copy and paste logging statements into each handler.
 */
func logHttp(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := NewStatusResponseWriter(w)
        log.Println(r.Method, " ", r.URL, " ", r.Proto)
        handler(sw, r)
		dur := time.Since(start)
        took := float64(dur) / float64(time.Millisecond)
		log.Printf("    --> %d %s - %0.3fms\n", sw.statusCode, http.StatusText(sw.statusCode), took)
    }
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
    host := r.Host
    allowed := true

    u, err := url.Parse("http://www.but-i-digress.ca")
    if err != nil {
        log.Fatal(err)
    }

    r.Host = u.Host
    log.Printf("host is %s, proxying to %s\n", host, u.Host)

    if allowed {
        proxy := httputil.NewSingleHostReverseProxy(u)
        proxy.ServeHTTP(w, r)
    } else {
        w.Write([]byte("403: Host forbidden " + host))
    }
}

func main() {

    mux := http.NewServeMux()
    mux.HandleFunc("/", logHttp(defaultHandler))

    server := &http.Server {
        Addr:   listen,
        Handler: mux,
        ErrorLog: logger,
    }

    log.Printf("Starting server on %s...\n", listen)
    err := server.ListenAndServe()
    if err != nil {
        log.Fatal(err)
    }
}
