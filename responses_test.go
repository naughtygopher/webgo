package webgo

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSendHeader(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	SendHeader(w, http.StatusNoContent)
	if w.Result().StatusCode != http.StatusNoContent {
		t.Errorf("Expected code '%d', got '%d'", http.StatusNoContent, w.Result().StatusCode)
	}
}

func TestSendError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	payload := map[string]string{"message": "hello world"}
	SendError(w, payload, http.StatusBadRequest)

	resp := struct {
		Errors map[string]string
	}{}

	body, err := io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if !reflect.DeepEqual(payload, resp.Errors) {
		t.Errorf(
			"Expected '%v', got '%v'. Raw response: '%s'",
			payload,
			resp.Errors,
			string(body),
		)
	}
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusBadRequest,
			w.Result().StatusCode,
			string(body),
		)
	}

	// testing invalid response body
	w = httptest.NewRecorder()

	invResp := struct {
		Errors string
	}{}
	invalidPayload := make(chan int)
	SendError(w, invalidPayload, http.StatusBadRequest)
	body, err = io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}
	err = json.Unmarshal(body, &invResp)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if invResp.Errors != `Internal server error` {
		t.Errorf(
			"Expected 'Internal server error', got '%v'. Raw response: '%s'",
			invResp.Errors,
			string(body),
		)
	}

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusInternalServerError,
			w.Result().StatusCode,
			string(body),
		)
	}

}

func TestSendResponse(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	payload := map[string]string{"hello": "world"}

	SendResponse(w, payload, http.StatusOK)
	body, err := io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	resp := struct {
		Data map[string]string
	}{}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		t.Error(err.Error())
	}

	if !reflect.DeepEqual(payload, resp.Data) {
		t.Errorf(
			"Expected '%v', got '%v'. Raw response: '%s'",
			payload,
			resp.Data,
			string(body),
		)
	}

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusOK,
			w.Result().StatusCode,
			string(body),
		)
	}

	// testing invalid response payload
	w = httptest.NewRecorder()
	SendResponse(w, make(chan int), http.StatusOK)
	body, err = io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	invalidresp := struct {
		Errors string
	}{}

	err = json.Unmarshal(body, &invalidresp)
	if err != nil {
		t.Error(err.Error())
	}

	if !reflect.DeepEqual(`Internal server error`, invalidresp.Errors) {
		t.Errorf(
			"Expected '%v', got '%v'. Raw response: '%s'",
			payload,
			invalidresp.Errors,
			string(body),
		)
	}

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusInternalServerError,
			w.Result().StatusCode,
			string(body),
		)
	}
}

func TestSend(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	payload := map[string]string{"hello": "world"}
	reqBody, _ := json.Marshal(payload)

	Send(w, JSONContentType, string(reqBody), http.StatusOK)
	body, err := io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	resp := map[string]string{}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		t.Error(err.Error())
	}

	if !reflect.DeepEqual(payload, resp) {
		t.Errorf(
			"Expected '%v', got '%v'. Raw response: '%s'",
			payload,
			resp,
			string(body),
		)
	}

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusOK,
			w.Result().StatusCode,
			string(body),
		)
	}
}

func TestRender(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	data := struct {
		Hello string
	}{
		Hello: "world",
	}
	tpl := template.New("txttemp")
	tpl, err := tpl.Parse(`{{.Hello}}`)
	if err != nil {
		t.Error(err.Error())
		return
	}
	Render(w, data, http.StatusOK, tpl)

	body, err := io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}
	if w.Code != http.StatusOK {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusOK,
			w.Code,
			string(body),
		)
	}

	w = httptest.NewRecorder()
	invaliddata := 0

	tpl = template.New("invalid")
	tpl, err = tpl.Parse(`{{.Hello}}`)
	if err != nil {
		t.Error(err.Error())
		return
	}
	Render(w, invaliddata, http.StatusOK, tpl)

	body, err = io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	str := string(body)
	want := `Internal server error`
	if str != want {
		t.Errorf(
			"Expected '%s', got '%s'. Raw response: '%s'",
			want,
			str,
			str,
		)
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusInternalServerError,
			w.Code,
			string(body),
		)
	}
}

func TestResponsehelpers(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	want := "hello world"
	resp := struct {
		Data   string
		Errors string
		Status int
	}{}

	R200(w, want)

	body, err := io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if resp.Data != want {
		t.Errorf(
			"Expected '%s', got '%s'",
			want,
			resp.Data,
		)
	}
	if w.Code != http.StatusOK {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusOK,
			w.Code,
			string(body),
		)
	}

	// R201
	w = httptest.NewRecorder()
	resp.Data = ""
	R201(w, want)

	body, err = io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if resp.Data != want {
		t.Errorf(
			"Expected '%s', got '%s'",
			want,
			resp.Data,
		)
	}
	if w.Code != http.StatusCreated {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusCreated,
			w.Code,
			string(body),
		)
	}

	// R204
	w = httptest.NewRecorder()
	resp.Data = ""
	R204(w)

	body, err = io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}
	if string(body) != "" {
		t.Errorf(
			"Expected empty response, got '%s'",
			string(body),
		)
	}
	if w.Code != http.StatusNoContent {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusNoContent,
			w.Code,
			string(body),
		)
	}

	// R302
	w = httptest.NewRecorder()
	resp.Data = ""
	R302(w, want)

	body, err = io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if resp.Data != want {
		t.Errorf(
			"Expected '%s', got '%s'",
			want,
			resp.Data,
		)
	}
	if w.Code != http.StatusFound {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusFound,
			w.Code,
			string(body),
		)
	}

	// R400
	w = httptest.NewRecorder()
	resp.Data = ""
	resp.Errors = ""
	R400(w, want)

	body, err = io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if resp.Errors != want {
		t.Errorf(
			"Expected '%s', got '%s'",
			want,
			resp.Errors,
		)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusBadRequest,
			w.Code,
			string(body),
		)
	}

	// R403
	w = httptest.NewRecorder()
	resp.Data = ""
	resp.Errors = ""
	R403(w, want)

	body, err = io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if resp.Errors != want {
		t.Errorf(
			"Expected '%s', got '%s'",
			want,
			resp.Errors,
		)
	}
	if w.Code != http.StatusForbidden {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusForbidden,
			w.Code,
			string(body),
		)
	}

	// R404
	w = httptest.NewRecorder()
	resp.Data = ""
	resp.Errors = ""
	R404(w, want)

	body, err = io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if resp.Errors != want {
		t.Errorf(
			"Expected '%s', got '%s'",
			want,
			resp.Errors,
		)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusNotFound,
			w.Code,
			string(body),
		)
	}

	// R406
	w = httptest.NewRecorder()
	resp.Data = ""
	resp.Errors = ""
	R406(w, want)

	body, err = io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if resp.Errors != want {
		t.Errorf(
			"Expected '%s', got '%s'",
			want,
			resp.Errors,
		)
	}
	if w.Code != http.StatusNotAcceptable {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusNotAcceptable,
			w.Code,
			string(body),
		)
	}

	// R451
	w = httptest.NewRecorder()
	resp.Data = ""
	resp.Errors = ""
	R451(w, want)

	body, err = io.ReadAll(w.Body)
	if err != nil {
		t.Error(err.Error())
		return
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if resp.Errors != want {
		t.Errorf(
			"Expected '%s', got '%s'",
			want,
			resp.Errors,
		)
	}
	if w.Code != http.StatusUnavailableForLegalReasons {
		t.Errorf(
			"Expected response status code %d, got %d. Raw response: '%s'",
			http.StatusUnavailableForLegalReasons,
			w.Code,
			string(body),
		)
	}

}
