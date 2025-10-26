package server

import (
	"fmt"
	"fusoxy/pkg/constants"
)

type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
	Status  int         `json:"status"`
}

func NewResponse(message string, data interface{}, error string, status int) *Response {
	return &Response{Message: message, Data: data, Error: error, Status: status}
}

type ResponseError struct {
	Message string `json:"message"`
	Error   string `json:"error"`
	Status  int    `json:"status"`
}

type ProxyRessourceResponse struct {
	Message            string `json:"message"`
	FuseRessourceInfos struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"fuse_ressource_infos"`
}

type RemoteRessourceResponse struct {
	Message             string `json:"message"`
	ProxyRessourceInfos struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"proxy_ressource_infos"`
}

func NewProxyRessourceResponse(id string) *ProxyRessourceResponse {
	return &ProxyRessourceResponse{
		Message: "Fusible ressource created",
		FuseRessourceInfos: struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		}{ID: id, URL: fmt.Sprintf("%s/%s", constants.FUSE_RESSOURCE_HANDLER, id)},
	}
}

func NewRemoteRessourceResponse(id string, registeredURL string) *RemoteRessourceResponse {
	return &RemoteRessourceResponse{
		Message: "Proxy ressource created",
		ProxyRessourceInfos: struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		}{ID: id, URL: fmt.Sprintf("%s/%s", constants.PROXY_RESSOURCE_HANDLER, id)},
	}
}
