package server

import (
	"fmt"
	"fusoxy/pkg/constants"
	"fusoxy/pkg/db"
	"log"

	"github.com/labstack/echo/v4"
)

// echo boilerplate serve

var testURLs = []string{
	"https://www.google.com",
	"https://scrapfly.io/svg/logo.svg",
	"https://cdn1.suno.ai/acfc6f20-2559-4d61-b680-62c97086929c.mp3",
}

func StartServer() {
	rdb := db.NewRemoteRessourceDB()
	for _, url := range testURLs {
		p, err := rdb.GetOrSet(url)
		if err != nil {
			log.Fatalf("Failed to get or set remote ressource: %v", err)
		}
		fmt.Println("Remote ressource created: ", p.ID)
	}
	fusibleHandlers := NewFusibleHandlers(rdb)
	e := echo.New()
	e.GET(constants.FUSE_RESSOURCE_HANDLER+"/:fid", fusibleHandlers.HandleFuseProxy())
	e.GET(constants.PROXY_RESSOURCE_HANDLER+"/:rid", fusibleHandlers.HandleFusibleRequest())
	e.GET(constants.REMOTE_RESSOURCE_HANDLER+"/:rid", fusibleHandlers.HandleRemoteRessourceProxy())
	e.GET(constants.DISPOSABLE_RESSOURCE_HANDLER+"/:url", fusibleHandlers.HandleDisposableRessourceProxy())
	e.Logger.Fatal(e.Start(":8080"))
}
