package forest

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"sync"
)

type AppContext struct {
	Routes Routes
}

type HttpRequest struct {
	http.Request
	Route Route
}

type HttpResponse struct {
	http.ResponseWriter
	Body       []byte
	Headers    map[string]string
	statusCode int
}

func (httpResponse *HttpResponse) RedirectTo(location string) {
	httpResponse.statusCode = 302
	httpResponse.Headers["location"] = location
}

func (httpResponse *HttpResponse) SetStatusCode(statusCode int) {
	httpResponse.statusCode = statusCode
}

func (httpResponse *HttpResponse) GetStatusCode() int {
	return httpResponse.statusCode
}

type Routes []Route

type Route struct {
	Method  string
	Path    string
	Name    string
	Handler RouteActionHandler
}

type RouteActionHandler func(request *HttpRequest, response *HttpResponse) ActionResult

type Controller struct{}

type ActionResult interface {
	serve_request(request *HttpRequest, response *HttpResponse)
}

type httpStatus struct {
	StatusCode int
}

func HttpStatus(statusCode int) httpStatus {

	return httpStatus{
		StatusCode: statusCode,
	}
}

type rawResult struct {
	Status  httpStatus
	Content string
	Headers map[string]string
}

func (result rawResult) serve_request(request *HttpRequest, response *HttpResponse) {

	// write the headers out...
	for name, value := range result.Headers {
		response.ResponseWriter.Header().Set(name, value)
	}

	// write the status code...
	response.ResponseWriter.WriteHeader(int(result.Status.StatusCode))

	// write the body...
	response.ResponseWriter.Write([]byte(result.Content))
}

func RawResult(content string, statusCode httpStatus) rawResult {
	return rawResult{
		Status:  statusCode,
		Content: content,
		Headers: make(map[string]string, 0),
	}
}

func JsonResult(data interface{}, statusCode httpStatus) ActionResult {

	var json_data, _ = json.Marshal(data)
	json_string := fmt.Sprintf("%s", json_data)

	var result = RawResult(json_string, statusCode)
	result.Headers["Content-Type"] = "application/json;charset=utf-8"

	return result
}

func (ctx *AppContext) Initialise() {
	log.SetOutput(os.Stdout)
	ctx.Routes = make([]Route, 0)

}

func (ctx *AppContext) ListenAndServe(address string) {
	router := mux.NewRouter().StrictSlash(false)
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "404 Not Found")
	})

	for _, route := range ctx.Routes {

		var handler = request_handler(route)

		//handler = ctx.Logger.Attach(handler, route.Name)

		router.Methods(route.Method).
			Path(route.Path).
			Name(route.Name).
			Handler(handler)
	}
	go log.Fatal(http.ListenAndServe(address, router))
}

func request_handler(route Route) http.Handler {

	return context.ClearHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		request := HttpRequest{
			Request: *r,
			Route:   route,
		}
		response := HttpResponse{
			ResponseWriter: w,
			Headers:        make(map[string]string, 0),
		}

		result := route.Handler(&request, &response)

		result.serve_request(&request, &response)
	}))
}

var forest_context *AppContext
var once sync.Once

func Context() *AppContext {
	once.Do(func() {
		forest_context = &AppContext{}
	})
	return forest_context
}
