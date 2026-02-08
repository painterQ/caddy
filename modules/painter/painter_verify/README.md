插件通过实现Authenticator接口完成身份认证功能
type Authenticator interface {
    Authenticate(http.ResponseWriter, *http.Request) (User, bool, error)
}

caddyauth.User怎么使用？
1. 后续中间件或者handler，可以通过`user, authenticated := caddyauth.GetAuthenticatedUser(r.Context())`获取
2. 反向代理后的后端服务，需要通过header_up指令搭配自定义的占位符
3. 关于占位符的操作，见文档https://caddyserver.com/docs/conventions#placeholders


我需要的四个Header
"X-JWT-Name", "X-JWT-Email", "X-JWT-ID", "X-JWT-Role"

username,email,user_id,roles