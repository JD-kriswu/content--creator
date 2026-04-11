package handler

import (
	"net/http"
	"strconv"

	"content-creator-imm/internal/repository"
	"github.com/gin-gonic/gin"
)

// GetFeishuBots 获取用户绑定的飞书机器人列表
func GetFeishuBots(c *gin.Context) {
	userID := c.GetUint("userID")

	bots, err := repository.GetFeishuBotsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bots": bots})
}

// UnbindFeishuBot 解绑飞书机器人
func UnbindFeishuBot(c *gin.Context) {
	userID := c.GetUint("userID")
	botIDStr := c.Param("id")

	botID, err := strconv.ParseUint(botIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}

	if err := repository.DeleteFeishuBot(uint(botID), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解绑失败"})
		return
	}

	// 同时删除关联的飞书会话
	repository.DeleteFeishuConvsByBotID(uint(botID))

	c.JSON(http.StatusOK, gin.H{"message": "解绑成功"})
}