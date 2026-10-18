package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/config"
	"offer-hub/backend/src/data"
	rt "offer-hub/backend/src/router"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("startup failed: %v", err)
	}
}

// run 返回后再由 main 退出，确保初始化失败时已建立的连接得到清理。
func run() (err error) {
	step := "config.Init"
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%s panicked: %v", step, recovered)
		} else if err != nil {
			err = fmt.Errorf("%s: %w", step, err)
		}
	}()

	if err = config.Init(""); err != nil {
		return err
	}
	step = "data.NewData"
	db, err := data.NewData(&config.Conf)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("close databases: %v", closeErr)
		}
	}()
	if config.Conf.JWT.Enable || config.Conf.RateLimit.Enable {
		step = "data.InitRedis"
		if err = data.InitRedis(&config.Conf); err != nil {
			return err
		}
		defer func() {
			if closeErr := data.CloseRedis(); closeErr != nil {
				log.Printf("close Redis: %v", closeErr)
			}
		}()
	}

	step = "gin.Default"
	router := gin.Default()
	step = "rt.RegisterRouter"
	rt.RegisterRouter(router)
	step = "router.Run"
	return router.Run(":" + strconv.Itoa(config.Conf.Common.Port))
}
