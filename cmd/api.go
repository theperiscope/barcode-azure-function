package main

import (
	"context"
	"fmt"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/ean"
	"github.com/boombuler/barcode/pdf417"
)

// based on https://github.com/benhoyt/go-routing/blob/master/retable/route.go
// and excellent https://benhoyt.com/writings/go-routing/ article
type route struct {
	method  string
	regex   *regexp.Regexp
	handler http.HandlerFunc
}

func newRoute(method, pattern string, handler http.HandlerFunc) route {
	return route{method, regexp.MustCompile("^" + pattern + "$"), handler}
}

var routes = []route{
	newRoute("GET", "/favicon.ico", faviconNoContent),
	newRoute("GET", "/barcode/(code39|code128|ean|pdf417)/([0-9]+)x([0-9]+)/([^/]+)\\.(gif|jpg|png)", getBarcode),
}

func getBarcode(w http.ResponseWriter, r *http.Request) {
	barcodeType := getField(r, 0)
	width, _ := strconv.Atoi(getField(r, 1))
	height, _ := strconv.Atoi(getField(r, 2))
	text := getField(r, 3)
	imageType := getField(r, 4)

	pdf417SecurityLevel := int(0)
	if barcodeType == "pdf417" {
		x := r.URL.Query().Get("securityLevel")
		if x != "" {
			pdf417SecurityLevel, _ = strconv.Atoi(x)
		}
	}

	etag := fmt.Sprintf("\"%s|%d|%d|%d|%s|%s\"", barcodeType, width, height, pdf417SecurityLevel, text, imageType)

	// cache hit
	if match := r.Header.Get("If-None-Match"); match != "" {
		if match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	var (
		img barcode.Barcode
		err error
	)

	switch barcodeType {
	case "code128":
		img, err = code128.Encode(text)
	case "code39":
		img, err = code39.Encode(text, true, false)
	case "ean":
		img, err = ean.Encode(text)
	case "pdf417":
		img, err = pdf417.Encode(text, byte(pdf417SecurityLevel))
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	img, _ = barcode.Scale(img, width, height)

	w.Header().Set("Etag", etag)
	w.Header().Set("Cache-Control", "max-age=2592000") // 30 days

	switch imageType {
	case "gif":
		w.Header().Set("Content-Type", "image/gif")
		gif.Encode(w, img, &gif.Options{NumColors: 256})
	case "jpg":
		w.Header().Set("Content-Type", "image/jpeg")
		jpeg.Encode(w, img, &jpeg.Options{Quality: 90})
	case "png":
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, img)
	}
}

type ctxKey struct{}

func getField(r *http.Request, index int) string {
	fields := r.Context().Value(ctxKey{}).([]string)
	return fields[index]
}

func Serve(w http.ResponseWriter, r *http.Request) {

	var allow []string
	for _, route := range routes {
		matches := route.regex.FindStringSubmatch(r.URL.Path)
		if len(matches) > 0 {
			if r.Method != route.method {
				allow = append(allow, route.method)
				continue
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, matches[1:])
			route.handler(w, r.WithContext(ctx))
			return
		}
	}
	if len(allow) > 0 {
		w.Header().Set("Allow", strings.Join(allow, ", "))
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.NotFound(w, r)
}

func faviconNoContent(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "No content", http.StatusNoContent)
}

func getPort() string {
	port := ":8080"
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		port = ":" + val
	}
	return port
}

func main() {
	port := getPort()
	srv := &http.Server{Addr: port, Handler: http.HandlerFunc(Serve)}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Printf("Server Started on port %s", port)

	<-done
	log.Print("Server Stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		// extra handling here
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Print("Server Exited Properly")
}
