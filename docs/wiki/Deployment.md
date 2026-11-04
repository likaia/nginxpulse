# 部署方式

## 版本要求
- 版本 > 1.5.3 单体部署时，必须搭建好 PostgreSQL，SQLite 已弃用。

## Docker 单容器（内置 PostgreSQL）
镜像内已集成 PostgreSQL，推荐此方式。

一键启动（极简配置，首次启动进入初始化向导）：
```bash
docker run -d --name nginxpulse \
  -p 8088:8088 \
  -e PUID=1000 \
  -e PGID=1000 \
  -v ./docker_local/logs:/share/logs:ro \
  -v ./docker_local/nginxpulse_data:/app/var/nginxpulse_data \
  -v ./docker_local/pgdata:/app/var/pgdata \
  -v ./docker_local/configs:/app/configs \
  -v /etc/localtime:/etc/localtime:ro \
  magiccoders/nginxpulse:latest
```
> PUID/PGID 需要替换为你本机所对应的正确值，此处的作用是用于应对目录权限不一致时引发的应用无法读取日志的情况。具体请移步下方的**Docker 部署权限说明**)

常用环境变量（容器内置 PG）：
- `POSTGRES_USER`/`POSTGRES_PASSWORD`/`POSTGRES_DB`: PG 账号与库名
- `POSTGRES_PORT`: PG 端口（默认 5432）
- `POSTGRES_LISTEN`: PG 监听地址（默认 127.0.0.1）
- `POSTGRES_CONNECT_HOST`: 应用连接 PG 的地址（默认 127.0.0.1）
- `DATA_DIR`: 数据目录（默认 `/app/var/nginxpulse_data`）
- `PGDATA`: PG 数据目录（默认 `/app/var/pgdata`）

如果你想外接自建 PG，可显式传入 `DB_DSN`，内置 PG 会被绕过。
此时不会启动内置 PG，`POSTGRES_*` 参数会被忽略，`/app/var/pgdata` 也无需挂载。


### 数据库账号权限与旧版本升级

推荐使用专用普通账号管理应用数据库内的表、索引、序列和分区，避免拥有 `SUPERUSER`、`CREATEDB`、`CREATEROLE`、`REPLICATION`、`BYPASSRLS`，以及通过角色成员关系获得这些权限或 `pg_read_server_files`、`pg_write_server_files`、`pg_execute_server_program`。应用启动时会检查这些权限；发现权限过高或无法检查时仅记录告警，仍沿用现有连接，不阻止升级启动。

内置 PostgreSQL 在启动后、启动后端前自动完成账号分离。新安装使用独立的 `POSTGRES_ADMIN_USER`（默认 `nginxpulse_admin`）初始化数据库，应用仍使用 `POSTGRES_USER`。升级已有数据目录时，会将原来的引导超级用户改名为管理账号，再以原应用账号名创建普通账号并迁移应用数据库内的表和序列所有权；数据库内容和应用连接账号名保持不变，重复启动可安全重试。管理账号清除密码，仅用于容器内 Unix socket 管理，不提供给应用。`POSTGRES_ADMIN_USER` 必须与 `POSTGRES_USER` 不同。升级前请备份 `pgdata`；升级后不要直接换回旧镜像（旧入口依赖应用账号的建库权限）。

外接数据库升级时无需修改现有账号或 `DB_DSN`。应用不会自动重命名、降权或修改外接账号及其角色成员关系；两处 SQL 注入修复不依赖数据库降权。若希望进一步收紧权限，可由数据库管理员创建专用普通账号，将应用数据库及已有应用表、序列的所有权交给该账号，再更新 `DB_DSN`（可选加固，不是升级前置条件）。若原账号是 `initdb` 创建的引导超级用户，PostgreSQL 不允许直接将它降为普通账号，应创建新账号。对于专用应用数据库，可使用仓库内的脚本：

```bash
# 使用管理员连接创建普通应用账号及数据库；请替换示例连接及密码。
psql 'postgresql://admin@db-host/postgres' -v ON_ERROR_STOP=1 \
  -v app_user=nginxpulse_app -v app_password='replace-with-a-strong-password' \
  -v app_db=nginxpulse -f scripts/postgres-app-role.sql
# 在应用数据库内，将原属主拥有的应用表/序列/视图及 schema 交给新账号。
psql 'postgresql://admin@db-host/nginxpulse' -v ON_ERROR_STOP=1 \
  -v app_user=nginxpulse_app -v admin_user=original_owner \
  -f scripts/postgres-app-owner.sql
```

安全回归测试：`bash scripts/test_security.sh`。脚本使用并删除独立的临时 PostgreSQL 容器，验证两处注入、HTTP 错误脱敏、新库初始化、旧库升级、普通账号的建表/分区/清理功能、内置账号的服务器文件读写和程序执行权限被拒绝，以及外接高权限账号仅告警、不改权限并可继续使用。

## Docker Compose
仓库根目录已提供 `docker-compose.yml`，可直接复制修改：
- 调整日志挂载路径。
- 若日志或数据目录权限不一致，可配置 `PUID/PGID` 对齐宿主机 UID/GID。
- 挂载 `nginxpulse_data`、`configs` 保持数据持久化；如使用内置 PG 再挂载 `pgdata`。
- 保持 `/etc/localtime` 只读挂载，以确保时区一致。
- SELinux 系统（RHEL/CentOS/Fedora）可在 volume 后追加 `:z` 或 `:Z` 重新打标签。

一键启动（极简配置，首次启动进入初始化向导）：
```bash
docker compose -f docker-compose-simple.yml up -d
```

## Docker 部署权限说明

镜像默认以非 root 用户（`nginxpulse`）运行。容器里能否读取日志、写入数据，**取决于宿主机目录的权限**。你在容器里用 `cat` 看到日志，通常是因为 `docker exec` 默认是 root，不代表应用用户有权限。

推荐做法：**让容器内用户的 UID/GID 与宿主机日志/数据目录的属主一致**。

步骤 1：查看宿主机目录的 UID/GID
```bash
ls -n /path/to/logs /path/to/nginxpulse_data /path/to/pgdata
# 或
stat -c '%u %g %n' /path/to/logs /path/to/nginxpulse_data /path/to/pgdata
```

步骤 2：启动容器时传入 `PUID/PGID`（与上面一致）
```bash
docker run ... \
  -e PUID=1000 \
  -e PGID=1000 \
  -v /path/to/logs:/var/log/nginx:ro \
  -v /path/to/nginxpulse_data:/app/var/nginxpulse_data:rw \
  -v /path/to/pgdata:/app/var/pgdata:rw \
  ...
```

步骤 3：确保目录对该 UID/GID 可读/可写
```bash
chown -R 1000:1000 /path/to/nginxpulse_data /path/to/pgdata
chmod -R u+rx /path/to/logs
```

如果你使用外部数据库（设置 `DB_DSN`），可以不挂载 `pgdata`。

SELinux 说明（RHEL/CentOS/Fedora 等）：
- 这些系统默认启用 SELinux，Docker 挂载目录可能因安全上下文导致“看得见但不可访问”。
- 解决办法是在 volume 后加 `:z` 或 `:Z` 重新打标签：
  - `:Z` 让该目录仅供当前容器使用（更严格）。
  - `:z` 让该目录可被多个容器共享使用。
```bash
docker run ... \
  -v /path/to/logs:/var/log/nginx:ro,Z \
  -v /path/to/nginxpulse_data:/app/var/nginxpulse_data:rw,Z \
  -v /path/to/pgdata:/app/var/pgdata:rw,Z \
  ...
```

不推荐做法：直接 `chmod -R 777`。这虽然省事，但权限过宽不安全，仅建议临时排查时使用。

## 单体部署（非 Docker）
适用于裸机或自建服务环境。需要用户自行安装 PostgreSQL。

步骤建议：
1. 安装 PostgreSQL 并创建数据库与用户。
2. 配置 `configs/nginxpulse_config.json` 中的 `database.dsn`。
3. 启动服务（可使用 `scripts/build_single.sh` 构建后运行）。

`database.dsn` 示例：
```json
"database": {
  "driver": "postgres",
  "dsn": "postgres://nginxpulse:nginxpulse@127.0.0.1:5432/nginxpulse?sslmode=disable",
  "maxOpenConns": 10,
  "maxIdleConns": 5,
  "connMaxLifetime": "30m"
}
```

## 本地开发
使用 `scripts/dev_local.sh`：
- 默认启动本地 docker postgres（`nginxpulse-postgres`），数据落在 docker volume `nginxpulse_pgdata`。
- 如需全量重置：`docker volume rm nginxpulse_pgdata`。

## 端口说明
- 8088: 前端页面
- 8089: API 服务

## 自定义前端访问前缀
当需要以二级路径访问前端（如 `/nginxpulse/`）时，可通过系统配置 `system.webBasePath` 设置前缀。该前缀仅支持**单段路径**，修改后需要重启服务生效，且根路径 `/` 将不再可访问。

### Docker 镜像（内置 Nginx）
1. 在系统配置中设置 `webBasePath`（或环境变量 `WEB_BASE_PATH`）。
2. 重启容器，访问地址变为：
   - PC：`/<base>/`
   - Mobile：`/<base>/m/`
   - API：`/<base>/api/`

镜像会在启动时生成新的 Nginx 配置与 `/app-config.js`，无需重新构建前端。

### 单体部署（Go 内嵌前端）
1. 在配置文件中设置 `system.webBasePath`。
2. 重启服务后访问：
   - PC：`/<base>/`
   - Mobile：`/<base>/m/`
   - API：`/<base>/api/`

### 设计原理
1. 通过 `/app-config.js` 下发 `window.__NGINXPULSE_BASE_PATH__`，前端运行期读取基路径，避免重新构建。
2. 服务端中间件强制前缀隔离，只有 `/<base>/...` 能访问，API 仅允许 `/<base>/api/*`。
3. 静态资源与 `/app-config.js` 不绑定前缀，保证前端可正常加载。

## 时区设置
本项目使用系统时区进行日志解析与统计，请确保运行环境时区正确。
- Docker: 挂载 `/etc/localtime:/etc/localtime:ro`
- 裸机: 确保系统时区已配置（例如 `timedatectl set-timezone Asia/Shanghai`）

## UI配置化
本文档列举的所有配置，均可在[系统配置](https://nginx-pulse.kaisir.cn/settings)中进行可视化设置。
