MySQLApp实现并注册为 caddy.App，模块ID (caddy.sql_db) 也是顶层型的
Caddy 的 Caddyfile 适配器会把 app 模块映射为 Caddyfile 的顶层块


可以在其他模块的Provision方法中通过ctx.App("sql_db")获取sql_db实例

caddyfile中应该写在顶级模块中：
```caddyfile
{
    https_port 443
    default_bind 0.0.0.0
    admin off
    log {
            level debug  # 调试日志，方便看请求转发
            output stdout # 日志输出到控制台
        }
    sql_db {
        addr ./sqlite.db
        type sqlite
        dbname user
    }
}
#站点块
```