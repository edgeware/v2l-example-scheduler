package v2l

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

func httpRequest(domain, method, path string, reqBody io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, domain+path, reqBody)
	if err != nil {
		// jsonResponse(w, JSONMessage{"REQ ERROR"}, http.StatusBadRequest)
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// jsonResponse(w, JSONMessage{"CLIENT DO ERROR"}, http.StatusBadRequest)
		return nil, err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// jsonResponse(w, JSONMessage{"Can't read body"}, http.StatusBadRequest)
		return nil, err
	}

	if strings.Contains(string(respBody), "{\"message\":") {
		return nil, fmt.Errorf(string(respBody))
	}
	defer resp.Body.Close()

	return respBody, nil
}

func uploadJSON(server string, method, apiPath string, data interface{}) ([]byte, error) {
	jsAPs, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(jsAPs)
	resp, err := httpRequest(server, method, apiPath, buf)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
