package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/subhammurarka/GreedyGamesAssignment/Config"
	"github.com/subhammurarka/GreedyGamesAssignment/DBCore"
	"github.com/subhammurarka/GreedyGamesAssignment/Handler"
)

var DbObj *DBCore.DB

func setupFlags() {
	flag.StringVar(&Config.AppConfig.Host, "host", "0.0.0.0", "host for the DB")
	flag.IntVar(&Config.AppConfig.Port, "port", 8080, "port for the DB")
	flag.Parse()
}

func main() {
	setupFlags()
	log.Println("Charging the RAM... Ready to serve!")

	DbObj = DBCore.NewDB()
	defer DbObj.Close()

	h := Handler.NewHandler(DbObj)

	r := gin.Default()

	addr := fmt.Sprintf("%s:%d", Config.AppConfig.Host, Config.AppConfig.Port)

	r.POST("/command", h.CommandServe)

	r.Run(addr)
}
