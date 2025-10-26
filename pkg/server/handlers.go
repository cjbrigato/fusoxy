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

		var RemoteRuleSet *db.RemoteRessourceRuleSet
		if  c.Request().Method == "POST" {
			if err := c.Bind(RemoteRuleSet); err != nil {
				c.Logger().Error("Error binding remote rule set: ", err)
				return c.String(http.StatusBadRequest, "Invalid request: "+err.Error())
			}
		}

		fusibleRessource, err := db.UrlToFusibleRessource(fdb, url, RemoteRuleSet)
		if err != nil {
			return c.String(http.StatusBadRequest, "Invalid request: "+err.Error())
		}
		response := NewProxyRessourceResponse(fusibleRessource.ID)
		return c.JSON(http.StatusOK, response)
	}
}

func HandleRemoteRessourceFullRegistration(rdb *db.RemoteRessourceDB) echo.HandlerFunc {
	return func(c echo.Context) error {

		override := false

		remoteRuleSet := &db.RemoteRessourceRuleSet{}
		if err := c.Bind(remoteRuleSet); err != nil {
			c.Logger().Error("Error binding remote rule set: ", err)
			return c.String(http.StatusBadRequest, "Invalid request: "+err.Error())
		}

		rid := c.Param("rid")
		url := fmt.Sprintf("%s?%s", rid, c.Request().URL.RawQuery)
		proxyRessource, err := rdb.GetOrSet(url, remoteRuleSet, override)
		if err != nil {
			c.Logger().Error("Error getting or setting remote ressource: ", err)
			return c.String(http.StatusBadRequest, "Invalid request: "+err.Error())
		}
		c.Logger().Info("Remote ressource proxy response: ", proxyRessource.ID, " ", proxyRessource.URL())
		response := NewRemoteRessourceResponse(proxyRessource.ID, proxyRessource.URL())
		return c.JSON(http.StatusOK, response)
	}
}

func HandleRemoteRessourceProxy(rdb *db.RemoteRessourceDB) echo.HandlerFunc {
	return func(c echo.Context) error {
		rid := c.Param("rid")
		url := fmt.Sprintf("%s?%s", rid, c.Request().URL.RawQuery)
		proxyRessource, err := rdb.GetOrSet(url, nil, false)
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

		if fusible.RuleSet != nil {
			if !fusible.RuleSet.Validate(c.Request(), db.RequestTypeConsume) {
				return c.String(http.StatusForbidden, "Fusible ressource rule set not valid")
			}
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
		fusibleRessource, err := db.FusibleRessourceFromRessourceID(fdb, rdb, rid, c.Request())
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
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

func (h *FusibleHandlers) HandleRemoteRessourceFullRegistration() echo.HandlerFunc {
	return HandleRemoteRessourceFullRegistration(h.RemoteRessourceDB)
}

func NewFusibleHandlers(rdb *db.RemoteRessourceDB) *FusibleHandlers {
	return &FusibleHandlers{FusibleRessourceDB: db.NewFusibleRessourceDB(),
		RemoteRessourceDB: rdb,
	}
}
