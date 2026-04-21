package cmd

import "github.com/gin-gonic/gin"

func RunServer() {
	// TODO: Implement server command.
	r := gin.New()

	err := r.Run()
	if err != nil {
		return
	}
}
