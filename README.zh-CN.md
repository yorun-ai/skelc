# skelc

[![License](https://img.shields.io/github/license/yorun-ai/skelc)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/yorun-ai/skelc)](go.mod)
[![CI](https://github.com/yorun-ai/skelc/actions/workflows/ci.yml/badge.svg)](https://github.com/yorun-ai/skelc/actions/workflows/ci.yml)

[English](README.md) | **简体中文**

skelc 是 Skel 契约语言的编译器和命令行工具。你可以用 `.skel` 文件统一描述 Vine 应用的领域数据、服务、事件和入口，然后生成 Go 服务端代码、TypeScript client 以及可供其他 domain 使用的公开契约。

如果你正在开发 Vine 应用，skelc 可以帮助你：

- 让服务端和客户端共享同一份类型与服务契约
- 在生成代码前发现语法、命名、类型和跨 domain 引用错误
- 生成可直接加入项目的 Go module 和 TypeScript client
- 只发布标记为 `pub` 的领域边界
- 格式化和校验契约
- 为 Binary 参数自动生成 vRPC 传输信息，以便应用使用 CBOR 高效传输二进制数据

## 安装

需要 Go 1.26 或更高版本：

```bash
go install go.yorun.ai/skelc/cmd/skelc@latest
skelc version
```

## 五分钟上手

创建 `user.skel`：

```skel
domain demo.user

pub actor ClientActor {
    via client {}
}

pub data User {
    id: int
    name: string
}

pub service UserService {
    for ClientActor via client

    method getUser {
        input {
            userId: int
        }
        output User?
    }
}
```

这里定义了一个可以通过 client 调用的 `UserService`。`pub` 只表示对应声明可以进入公开生成输出，不代表网络接口可以匿名访问。

先检查并格式化契约：

```bash
skelc check --skel-in ./user.skel
skelc format --skel-in ./user.skel
```

为 Go 服务端生成一个独立 module：

```bash
skelc gen go-module \
  --skel-in ./user.skel \
  --go-out ./generated/user-go \
  --go-module example.com/generated/user
```

为 TypeScript 应用生成类型和 service client：

```bash
skelc gen ts \
  --skel-in ./user.skel \
  --ts-out ./generated/user-ts
```

TypeScript 输出包含：

```text
generated/user-ts/
├── data.ts       # data 和 enum 类型
├── spec.ts       # vRPC service 描述
├── service.ts    # service client factory
└── index.ts      # 统一导出
```

把生成目录加入 TypeScript 项目后，可以使用已经配置好的 `VrpcClient` 创建服务：

```ts
import { createUserService } from './generated/user-ts';

const userService = createUserService(client);
const user = await userService.getUser({ userId: 1001 });
```

生成命令默认会先清理目标输出目录，因此输出目录应由 skelc 独占。只有明确需要保留其中其他文件时才使用 `--no-clean`。

仓库中的 [`examples/quickstart`](examples/quickstart) 提供了这套流程的可运行版本。检出仓库后，可以用下面的命令校验契约并生成全部支持的目标：

```bash
./examples/quickstart/generate.sh
```

## 常用工作流

### 使用目录管理一个 domain

契约变多后，可以把同一 domain 拆成多个文件：

```text
skel/
├── domain.skel
├── actor.skel
├── data.skel
└── service.skel
```

`domain.skel` 只能包含 domain 声明和可选的 `@desc`；其他文件也必须以相同的 domain 声明开头。此时将整个目录传给 `--skel-in`：

```bash
skelc check --skel-in ./skel
```

### 生成公开契约

将标记为 `pub` 的声明提取为可共享的 Skel：

```bash
skelc gen skel \
  --pub \
  --skel-in ./skel \
  --skel-out ./generated/public-skel
```

TypeScript 生成也支持 `--pub`，只输出公开 data、enum 和符合条件的 service client。

### 引用其他 domain

在 `.skel` 中声明 `import` 后，通过可重复使用的 `--skel-import domain=PATH` 指定外部契约位置。生成 Go module 或 TypeScript 时，再使用对应的 `--go-import`、`--go-module-prefix` 或 `--ts-import` 映射目标语言的 package。完整示例见 [CLI 参考](https://yorun.ai/skelc/cli)。

### 查询和格式化

```bash
skelc symbol list --skel-in ./skel
skelc symbol get demo.user.User --skel-in ./skel
skelc format --skel-in ./skel
```

`format` 会原地修改文件，执行前会先验证全部输入。工具集成可以使用全局参数 `--log-format jsonl` 获取机器可读诊断。

## 程序调用 API

Go 程序可以通过根 package `go.yorun.ai/skelc` 调用生成能力，无需导入实现 package：

```go
result, err := skelc.CompileGolang(
	skelc.Input{
		SkelIn: "./skel",
		SkelImports: map[string]string{
			"shared.types": "../shared/skel",
		},
	},
	skelc.GolangOption{
		Out:         "./generated/user-go",
		AsModule:    true,
		Module:      "example.com/generated/user",
		VineVersion: skelc.DefaultGolangVineVersion,
	},
)
if err != nil {
	return err
}
for _, warning := range result.Warnings {
	log.Print(warning)
}
```

API 同时提供 `CompileTypeScript` 和 `CompileSkeleton`。编译入口会先完成全部输入的校验和解析，之后才清理输出目录；如需保留已有文件，可在目标选项中设置 `NoClean`。

自定义 generator 可以调用 `skelc.Parse`，并通过与 parser 无关的 `go.yorun.ai/skelc/model` 使用返回的 `*model.Domain`。解析完成的模型已经包含由 skelc 计算好的兼容性 hash。内置的 `GenerateGolang`、`GenerateTypeScript` 和 `GenerateSkeleton` 也接受同一个已解析 domain，因此多个目标可以共享一次解析结果。

## skelc 与 Vine、vRPC

skelc 负责读取契约并生成代码，本身不是应用运行时：

- 生成的 Go 代码使用 `go.yorun.ai/vine` 提供的运行时类型和服务基础设施
- 生成的 TypeScript service client 使用 `@yorun-ai/vrpc`
- CBOR codec 等运行时能力由应用在创建 vRPC client 时配置，不会被 skelc 打包进生成代码

升级 skelc 后，应重新生成代码，并在使用方运行类型检查和测试。Skel 语法、CLI 行为、诊断格式以及生成 API 都属于兼容性边界。

## 命令概览

```text
check          校验 Skel 定义
format         原地格式化 Skel 定义
symbol         列出或查询顶层符号
gen skel       生成公开 Skel 契约
gen go         在现有 Go module 中生成代码
gen go-module  生成独立 Go module
gen ts         生成 TypeScript 类型和 client
version        显示 skelc 与默认 Vine 版本
```

运行 `skelc --help` 或 `skelc <command> --help` 查看当前版本支持的全部参数。

## 文档

- [Skel 语法参考](https://yorun.ai/skelc/syntax)
- [CLI 参考](https://yorun.ai/skelc/cli)
- [TypeScript 生成说明](https://yorun.ai/skelc/generation/typescript)
- [变更记录](CHANGELOG.md)（英文）
- [文档站源码](https://github.com/yorun-ai/vine-doc)
- [VS Code 扩展](tool/vscode-skel/README.md)（英文）

## 参与贡献

开发流程见 [CONTRIBUTING.md](CONTRIBUTING.md)，仓库结构和开发约定见 [AGENTS.md](AGENTS.md)。

skelc 遵循[语义化版本](https://semver.org/lang/zh-CN/)，并使用 [Apache License 2.0](LICENSE) 开源。
