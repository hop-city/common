package server

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func Parse(r *http.Request, expected interface{}) error {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	if expected == nil {
		return nil
	}
	if out, ok := expected.(*string); ok {
		*out = string(data)
	} else if out, ok := expected.(map[string]string); ok {
		params, err := url.ParseQuery(string(data))
		if err != nil {
			return errors.Errorf("rest/server.Parse: error parsing urlencoded data - '%s'", data)
		}
		for k, v := range params {
			out[k] = strings.Join(v, ",")
		}
	} else {
		err := json.Unmarshal(data, expected)
		if err != nil {
			return errors.Errorf("rest/server.Parse: error unmarshalling JSON data - '%s'", data)
		}
	}

	return nil
}
