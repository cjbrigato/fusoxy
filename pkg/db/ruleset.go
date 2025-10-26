package db

import (
	"encoding/base64"
	"net/http"
	"strings"
)

type RequestType string

const (
	RequestTypeRequest RequestType = "REQUEST"
	RequestTypeConsume RequestType = "CONSUME"
)

type BasicAuthCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type QueryParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// enforced when creating a fusible ressource
type RequestRuleSet struct {
	RequiredHeaders  http.Header `json:"required_headers,omitempty"`
	ValidatedHeaders http.Header `json:"validate_headers_values,omitempty"`

	RequiredQueryParams  []string     `json:"required_query_params,omitempty"`
	ValidatedQueryParams []QueryParam `json:"validate_query_params,omitempty"`

	AllowedOrigins []string `json:"allowed_origins,omitempty"`

	BasicAuthCredentials *BasicAuthCredentials `json:"basic_auth_credentials,omitempty"`
}

// enforced when consuming a fusible ressource
type FusibleRessourceRuleSet struct {
	*RequestRuleSet `json:"request_rule_set,omitempty"`

	AllowedMethods []string `json:"allowed_methods,omitempty"`

	ForwardHeaders     []string `json:"forward_headers,omitempty"`
	ForwardQueryParams bool     `json:"forward_query_params,omitempty"`
}

type RemoteRessourceRuleSet struct {
	RequestRuleSet          *RequestRuleSet          `json:"request_rule_set,omitempty"`
	FusibleRessourceRuleSet *FusibleRessourceRuleSet `json:"fusible_ressource_rule_set,omitempty"`
}

func (r *RemoteRessourceRuleSet) Validate(req *http.Request, reqType RequestType) bool {
	if reqType == RequestTypeRequest {
		if r.RequestRuleSet != nil {
			return r.RequestRuleSet.Validate(req)
		}
	} else if reqType == RequestTypeConsume {
		if r.FusibleRessourceRuleSet != nil {
			return r.FusibleRessourceRuleSet.Validate(req)
		}
	}
	return true
}

func (r *RequestRuleSet) Validate(req *http.Request) bool {
	validateBasicAuthCredentials := true
	validateRequiredHeaders := true
	validateValidatedHeaders := true
	validateRequiredQueryParams := true
	validateValidatedQueryParams := true
	validateAllowedOrigins := true

	if r.BasicAuthCredentials != nil {
		validateBasicAuthCredentials = r.BasicAuthCredentials.Validate(req)
	}
	if r.RequiredHeaders != nil {
		for _, header := range r.RequiredHeaders {
			if req.Header.Get(header[0]) == "" {
				validateRequiredHeaders = false
				break
			}
		}
	}
	if r.ValidatedHeaders != nil {
		for _, header := range r.ValidatedHeaders {
			if req.Header.Get(header[0]) == "" {
				validateValidatedHeaders = false
				break
			}
			if req.Header.Get(header[0]) != header[1] {
				validateValidatedHeaders = false
				break
			}
		}
	}
	if r.RequiredQueryParams != nil {
		for _, queryParam := range r.RequiredQueryParams {
			if req.URL.Query().Get(queryParam) == "" {
				validateRequiredQueryParams = false
				break
			}
		}
	}
	if r.ValidatedQueryParams != nil {
		for _, queryParam := range r.ValidatedQueryParams {
			if req.URL.Query().Get(queryParam.Name) == "" {
				validateValidatedQueryParams = false
				break
			}
			if req.URL.Query().Get(queryParam.Name) != queryParam.Value {
				validateValidatedQueryParams = false
				break
			}
		}
	}
	if r.AllowedOrigins != nil {
		for _, origin := range r.AllowedOrigins {
			if req.Header.Get("Origin") == origin {
				validateAllowedOrigins = true
				break
			}
		}
		validateAllowedOrigins = false
	}
	return validateBasicAuthCredentials && validateRequiredHeaders && validateValidatedHeaders && validateRequiredQueryParams && validateValidatedQueryParams && validateAllowedOrigins
}

func (r *FusibleRessourceRuleSet) Validate(req *http.Request) bool {
	validateRequestRuleSet := true

	if r.RequestRuleSet != nil {
		validateRequestRuleSet = r.RequestRuleSet.Validate(req)
	}

	validateAllowedMethods := true
	if r.AllowedMethods != nil {
		for _, method := range r.AllowedMethods {
			if req.Method == method {
				validateAllowedMethods = true
				break
			}
		}
		validateAllowedMethods = false
	}
	if r.ForwardHeaders != nil {
		for _, header := range r.ForwardHeaders {
			req.Header.Add(header, req.Header.Get(header))
		}
	}
	if r.ForwardQueryParams {
		req.URL.RawQuery = req.URL.RawQuery + "&" + req.URL.Query().Encode()
	}
	return validateRequestRuleSet && validateAllowedMethods
}

func (c *BasicAuthCredentials) Validate(req *http.Request) bool {
	// get Authorization basic
	authorization := req.Header.Get("Authorization")
	if authorization == "" {
		return false
	}
	parts := strings.Split(authorization, " ")
	if len(parts) != 2 {
		return false
	}
	if parts[0] != "Basic" {
		if parts[1] != base64.StdEncoding.EncodeToString([]byte(c.Username+":"+c.Password)) {
			return false
		}
	}
	return true
}
