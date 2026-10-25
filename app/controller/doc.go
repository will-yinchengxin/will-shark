// Package controller 可选的控制器层。
//
// 大多数接口直接在 router 中用 httpx.Handle 绑定 service 方法即可；
// 需要直接操作 gin.Context（读写 header/cookie、文件上传下载、SSE 流式输出等）时才使用 controller。
// 生成：go tool willshark-gen module <name> -with router,controller,service,dto
package controller
