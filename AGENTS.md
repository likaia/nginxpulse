## Git 提交规范

### Commit Message 格式

本仓库 commit message **优先使用 Conventional Commits** 规范，与历史风格保持一致。格式：

```
<type>(<scope>): <subject>        # 中文
<type>(<scope>): <subject>        # 英文
```

### Type 类型

| Type       | 用途                                 | 示例                                          |
| ---------- | ------------------------------------ | --------------------------------------------- |
| `feat`     | 新增服务/新功能                      | `feat: add adminer docker deployment files`  |
| `fix`      | 修复 bug                             | `fix(gitea): 修复备份脚本权限问题`            |
| `docs`     | 文档变更（readme、AGENTS.md 等）     | `docs: 添加git提交信息规则文件`               |
| `refactor` | 重构（不改功能）                     | `refactor(status.py): 重构OLED状态显示代码`  |
| `perf`     | 性能优化                             | `perf(redis): 调整内存淘汰策略`              |
| `build`    | 构建相关（Dockerfile、compose、依赖）| `build(moltbot): 添加生产环境docker compose` |
| `chore`    | 杂项（版本号、配置、镜像标签）       | `chore: 更新 Joplin 服务器镜像版本至 3.6.1`  |
| `style`    | 格式调整（不影响代码逻辑）           | `style: 统一 yaml 缩进为 2 空格`             |
| `test`     | 测试相关                             | `test: 添加 memos 部署验证脚本`              |
