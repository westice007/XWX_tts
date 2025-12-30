// 普通话拼音服务,供python训练端使用，两端统一

package main


import (
	"net/http"
	"time"
	 
	"github.com/gin-gonic/gin"
)

// UserRequest 定义接收的参数结构
type UserRequest struct {
	Zhtext string `json:"zhtext" binding:"required"` // binding:"required" 用于自动校验非空
}

func main() {
	// 1. 设置为生产模式以提升性能（隐藏调试日志）
	gin.SetMode(gin.ReleaseMode)

	// 2. 初始化引擎，使用 Gin 默认中间件（Logger 和 Recovery）
	r := gin.New()
	r.Use(gin.Recovery()) // 崩溃时自动恢复，保证服务器高可用

	// 3. 定义 POST 接口
	r.POST("/api/mandaren_pinyin", func(c *gin.Context) {
		var req UserRequest

		// 4. 将 JSON 参数绑定到结构体，若解析失败返回 400
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "参数格式错误",
				"details": err.Error(),
			})
			return
		}

		// 6. 调用 Mandaren_pinyin 函数处理中文文本
		pinyins := Mandaren_pinyin(req.Zhtext)

		// 5. 返回处理结果（JSON 字典）
		c.JSON(http.StatusOK, gin.H{
			"status":    "success",
			"message":   "数据接收成功",
			"timestamp": time.Now().Unix(),
			"data": gin.H{
				"zhtext": req.Zhtext,
				"pinyins": pinyins,
			},
		})
	})

	// 6. 启动服务器（默认 8080 端口）
	r.Run(":18484")
}