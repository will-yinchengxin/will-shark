// Package controller 管理 HTTP 输入输出：绑定和校验请求、调用 Service、映射错误并输出响应。
//
// 所有业务 HTTP 路由都绑定 Controller；Swagger 注释也放在这里。
// Service 使用普通 context.Context 和 app/do 中的 DTO，可被定时任务等入口复用。
package controller
