package server

import (
	"fusoxy/pkg/constants"
	"fusoxy/pkg/db"

	"github.com/labstack/echo/v4"
)

// echo boilerplate serve
func StartServer() {
	rdb := db.NewRemoteRessourceDB()
	fusibleHandlers := NewFusibleHandlers(rdb)
	e := echo.New()
	e.GET(constants.FUSE_RESSOURCE_HANDLER+"/:fid", fusibleHandlers.HandleFuseProxy())
	e.GET(constants.PROXY_RESSOURCE_HANDLER+"/:rid", fusibleHandlers.HandleFusibleRequest())
	e.GET(constants.REMOTE_RESSOURCE_HANDLER+"/:rid", fusibleHandlers.HandleRemoteRessourceProxy())
	e.GET(constants.DISPOSABLE_RESSOURCE_HANDLER+"/:url", fusibleHandlers.HandleDisposableRessourceProxy())
	e.POST(constants.REMOTE_RESSOURCE_HANDLER+"/:rid", fusibleHandlers.HandleRemoteRessourceFullRegistration())
	e.POST(constants.DISPOSABLE_RESSOURCE_HANDLER+"/:url", fusibleHandlers.HandleDisposableRessourceProxy())
	e.Logger.Fatal(e.Start(":8080"))
}

/*
```
// Exemple curl for full registration
curl -X POST "http://localhost:8080/remote/https://httpbin.dev/anything?meh=lol" -H "Content-Type: application/json" -d '{
    "request_rule_set": {
        "basic_auth_credentials": {
            "username": "admin",
            "password": "password"
        }
    }
}'
```
*/

/*
// Exemple curl for disposable registration

```
curl -X POST "http://localhost:8080/disposable/https://httpbin.dev/anything?meh=lol" \
-H "Content-Type: application/json" \
-d '{
	"fusible_ressource_rule_set": {
		"allowed_methods": ["POST"]
		"basic_auth_credentials": {
			"username": "admin",
			"password": "password"
		}
    }
}'
```
*/
