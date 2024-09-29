package main

import (
	"log"

	"github.com/joho/godotenv"	
	"github.com/m5m/scheduler"
	"github.com/m5m/api"
	"github.com/gin-gonic/gin"
	"github.com/m5m/discovery"
	"github.com/go-redis/redis/v8"
	"os"
	"github.com/m5m/model"
	"net/http"
)

func main() {
	r := gin.Default()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	rdb := redis.NewClient(&redis.Options{
        Addr: os.Getenv("REDIS_HOST"),
    })
	defer rdb.Close()
	nodes := scheduler.NewNodes(rdb)	
	discoveryService := discovery.NewDiscoveryService(rdb)	
	apiService := api.ApiServiceNew(discoveryService, nodes)	
	r.GET("/nodes", func(c *gin.Context) {
		c.JSON(http.StatusOK, apiService.GetNodes()) 
	})
	r.GET("/containers", func(c *gin.Context) {
		c.JSON(http.StatusOK, apiService.GetAllContainers())
	})
	r.POST("/apply", func(c *gin.Context) {
		var pod model.Pod
		c.BindJSON(&pod)
		container, err := apiService.Apply(pod)
		log.Default().Println(container)		
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, container)
		}
	})
	r.GET("/inspect/:containerName", func(c *gin.Context) {
		containerName := c.Param("containerName")
		c.JSON(http.StatusOK, apiService.Inspect(containerName))
	})
	r.DELETE("/remove/:podName", func(c *gin.Context) {
		containerName := c.Param("podName")
		err := apiService.Remove(containerName)
		if err != nil {
			c.JSON(http.StatusNotFound, err)
		} else {
			c.JSON(http.StatusOK, "Pod removed")
		}
	})	
	r.GET("/containers/:podName", func(c *gin.Context) {
		podName := c.Param("podName")
		result := apiService.GetContainersByPod(podName)
		if result == nil {
			c.JSON(http.StatusNotFound, "No containers found for pod " + podName)
			return
		}
		c.JSON(http.StatusOK, result)
	})
	r.DELETE("/container/:containerName", func(c *gin.Context) {
		containerName := c.Param("containerName")
		err := apiService.DeleteContainer(containerName)
		if err != nil {
			c.JSON(http.StatusNotFound, err)
		} else {
			c.JSON(http.StatusOK, "Container removed")
		}
	})
	r.GET("/restart/:containerName", func(c *gin.Context) {
		containerName := c.Param("containerName")
		err := apiService.RestartContainer(containerName)
		if err != nil {
			c.JSON(http.StatusNotFound, err)
		} else {
			c.JSON(http.StatusOK, "Container restarted")
		}
	})

	r.GET("/rebuild/:containerName", func(c *gin.Context) {
		containerName := c.Param("containerName")
		container, err := apiService.RebuildContainer(containerName)
		if err != nil {
			c.JSON(http.StatusNotFound, err)
		} else {
			c.JSON(http.StatusOK, container)
		}
	})

	r.Run(":3232")
	
}
