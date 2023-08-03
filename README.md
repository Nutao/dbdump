本工具用来导出MySQL的表结构定义

### 使用说明

```bash
NAME:
   dbdump - SQL Data Define Tool

USAGE:
   dbdump [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --dbType value, --DB value  db类型, 支持mysql，pgsql (default: "mysql")
   --host value, -h value      Connect to host. (default: "127.0.0.1")
   --port value, -P value      Port number to use for connection. (default: 3306)
   --user value, -u value      User for login if not current user. (default: "root")
   --password value, -p value  Password to use when connecting to server.
   --database value, -D value  Database to use.
   --tables value, -t value    Tables to get.
   --output value, -o value    Write to file instead of stdout.
   --format_type value         Format type of the output(gotext|json). (default: "json")
   --format_config value       Format config of the output. Filename prepend with @
   --help                      show help (default: false)
```

### 使用示例

```bash
[user@node] # 输出指定数据库所有的表定义
[user@node] $ dbdump -h 127.0.0.1 -P 3306 -u root -p password -D information_schema
[user@node] 
[user@node] # 指定多张表
[user@node] $ dbdump -h 127.0.0.1 -P 3306 -u root -p password -D information_schema -t TABLES -t TABLE_CONSTRAINTS
[user@node] 
[user@node] # 可以指定以GO模版形式输出（默认以json形式输出）
[user@node] $ dbdump -h 127.0.0.1 -P 3306 -u root -p password -D information_schema -t TABLES --format_type gotext --format_config "{{.}}"
[user@node] 
[user@node] # GO模板配置可以文件的形式给出
[user@node] $ dbdump -h 127.0.0.1 -P 3306 -u root -p password -D information_schema -t TABLES --format_type gotext --format_config "@assets/gotext_md.fc"
[user@node] 
[user@node] # 输出重定向到文件
[user@node] $ dbdump -h 127.0.0.1 -P 3306 -u root -p password -D information_schema -t TABLES --format_type gotext --format_config "@assets/gotext_md.fc" -o readme.md

# 指定DB类型输出
dbdump -DB mysql -h 127.0.0.1 -P 3306 -u root -p password -D information_schema -t TABLES --format_type gotext --format_config "@assets/gotext_md.fc" -o readme.md
```

