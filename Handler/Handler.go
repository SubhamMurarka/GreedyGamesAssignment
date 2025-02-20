package Handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/subhammurarka/GreedyGamesAssignment/DBCore"
	"github.com/subhammurarka/GreedyGamesAssignment/Models"
)

type Handler struct {
	DbObj *DBCore.DB
}

func NewHandler(DBObj *DBCore.DB) *Handler {
	return &Handler{
		DbObj: DBObj,
	}
}

func (h *Handler) CommandServe(c *gin.Context) {
	var req Models.Request

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Argument"})
		return
	}

	args := strings.Fields(req.Command)

	if err := Models.ValidateInput(args); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmdType := strings.ToUpper(args[0])

	if cmdType == "GET" {
		val, exists := h.DbObj.Get(args[1])
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"Value": val})
		return
	}

	if cmdType == "SET" {
		key := args[1]
		value := args[2]
		var expiry int
		nx := false
		xx := false

		// Parsing optional arguments
		for i := 3; i < len(args); i++ {
			switch strings.ToUpper(args[i]) {
			case "EX":
				seconds, _ := strconv.Atoi(args[i+1])
				if seconds > 0 {
					expiry = seconds
				} else {
					expiry = -1
				}
				i++
			case "NX":
				nx = true
			case "XX":
				xx = true
			}
		}

		// Directly call commands have validated the request above
		err := h.DbObj.Set(key, value, expiry, nx, xx)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "OK"})
		return
	}

	if cmdType == "QPUSH" {
		key := args[1]

		for _, value := range args[2:] {
			h.DbObj.Push(key, value)
		}

		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}

	if cmdType == "QPOP" {
		key := args[1]
		val, err := h.DbObj.Pop(key)
		if err != nil {
			if err == DBCore.ErrBlocked {
				c.JSON(http.StatusLocked, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"value": val})
		return
	}

	if cmdType == "BQPOP" {
		key := args[1]
		seconds, _ := strconv.ParseFloat(args[2], 64)
		val, err := h.DbObj.BQPOP(key, seconds)
		if err != nil {
			if err == DBCore.ErrBlocked {
				c.JSON(http.StatusLocked, gin.H{"error": err.Error()})
				return
			}
			if err == DBCore.ErrEmpty {
				c.JSON(http.StatusNoContent, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"value": val})
		return
	}
}
