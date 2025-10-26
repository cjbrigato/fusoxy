package server

import (
	"fmt"
	"fusoxy/pkg/db"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/labstack/echo/v4"
)

// one time use proxy ressource without registration
func HandleDisposableRessourceProxy(fdb *db.FusibleRessourceDB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("url")
		url := fmt.Sprintf("%s?%s", id, c.Request().URL.RawQuery)
		fusibleRessource, err := db.UrlToFusibleRessource(fdb, url)
		if err != nil {
			return c.String(http.StatusBadRequest, "Invalid request: "+err.Error())
		}
		response := NewProxyRessourceResponse(fusibleRessource.ID)
		return c.JSON(http.StatusOK, response)
	}
}

func HandleRemoteRessourceProxy(rdb *db.RemoteRessourceDB) echo.HandlerFunc {
	return func(c echo.Context) error {
		rid := c.Param("rid")
		url := fmt.Sprintf("%s?%s", rid, c.Request().URL.RawQuery)
		proxyRessource, err := rdb.GetOrSet(url)
		if err != nil {
			c.Logger().Error("Error getting or setting remote ressource: ", err)
			return c.String(http.StatusBadRequest, "Invalid request: "+err.Error())
		}
		c.Logger().Info("Remote ressource proxy response: ", proxyRessource.ID, " ", proxyRessource.URL())
		response := NewRemoteRessourceResponse(proxyRessource.ID, proxyRessource.URL())
		return c.JSON(http.StatusOK, response)
	}
}

func rewriteRequestURL(req *http.Request, target *url.URL) {
	targetQuery := target.RawQuery
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	req.URL.Path, req.URL.RawPath = target.Path, target.RawPath
	if targetQuery == "" || req.URL.RawQuery == "" {
		req.URL.RawQuery = targetQuery + req.URL.RawQuery
	} else {
		req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
	}
}

func SetURL(r *httputil.ProxyRequest, target *url.URL) {
	rewriteRequestURL(r.Out, target)
	r.Out.Host = ""
}

func HandleFuseProxy(fdb *db.FusibleRessourceDB) echo.HandlerFunc {
	return func(c echo.Context) error {
		fuseID := c.Param("fid")

		fusible, err := fdb.ConsumeFusibleRessource(fuseID)
		if err != nil || fusible == nil {
			return c.String(http.StatusInternalServerError, "Invalid or melted fuse ID")
		}

		fmt.Println("Fusible ressource consumed: ", fusible.ID, " for url: ", fusible.URL())

		targetURL, err := url.Parse(fusible.URL())
		if err != nil {
			return c.String(http.StatusInternalServerError, "Invalid URL")
		}
		proxy := &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				SetURL(r, targetURL)
			},
		}

		proxy.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

func HandleFusibleRequest(fdb *db.FusibleRessourceDB, rdb *db.RemoteRessourceDB) echo.HandlerFunc {
	return func(c echo.Context) error {
		rid := c.Param("rid")
		fusibleRessource, err := db.FusibleRessourceFromRessourceID(fdb, rdb, rid)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Invalid or melted remote ressource ID")
		}
		response := NewProxyRessourceResponse(fusibleRessource.ID)
		return c.JSON(http.StatusOK, response)
	}
}

type FusibleHandlers struct {
	FusibleRessourceDB *db.FusibleRessourceDB
	RemoteRessourceDB  *db.RemoteRessourceDB
}

func (h *FusibleHandlers) HandleFuseProxy() echo.HandlerFunc {
	return HandleFuseProxy(h.FusibleRessourceDB)
}

func (h *FusibleHandlers) HandleFusibleRequest() echo.HandlerFunc {
	return HandleFusibleRequest(h.FusibleRessourceDB, h.RemoteRessourceDB)
}

func (h *FusibleHandlers) HandleRemoteRessourceProxy() echo.HandlerFunc {
	return HandleRemoteRessourceProxy(h.RemoteRessourceDB)
}

func (h *FusibleHandlers) HandleDisposableRessourceProxy() echo.HandlerFunc {
	return HandleDisposableRessourceProxy(h.FusibleRessourceDB)
}

func NewFusibleHandlers(rdb *db.RemoteRessourceDB) *FusibleHandlers {
	return &FusibleHandlers{FusibleRessourceDB: db.NewFusibleRessourceDB(),
		RemoteRessourceDB: rdb,
	}
}
