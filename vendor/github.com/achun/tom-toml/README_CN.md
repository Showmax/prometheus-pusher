tom-toml
========

[TOML](https://github.com/mojombo/toml) 格式 Go 语言支持包.

本包支持 TOML 版本
[v0.2.0](https://github.com/mojombo/toml/blob/master/versions/toml-v0.2.0.md)

[![wercker status](https://app.wercker.com/status/28e2ac15ba6930f928b10187ad4043c3 "wercker status")](https://app.wercker.com/project/bykey/28e2ac15ba6930f928b10187ad4043c3)

## 文档

Go DOC 文档请访问
[gowalker.org](http://gowalker.org/github.com/achun/tom-toml).

[readme.toml](readme.toml) 是用 toml 格式对 tom-toml 的深入介绍.

## Import

    import "github.com/achun/tom-toml"


## 使用

假设你有一个TOML文件 `example.toml` 看起来像这样:

```toml
# 注释以"#"开头, 这是多行注释, 可以分多行
# tom-toml 把这两行注释绑定到紧随其后的 key, 也就是 title

title = "TOML Example" # 这是行尾注释, tom-toml 把此注释绑定给 title

# 虽然只有一行, 这也属于多行注释, tom-toml 把此注释绑定给 owner
[owner] # 这是行尾注释, tom-toml 把这一行注释绑定到 owner

name = "om Preston-Werner" # 这是行尾注释, tom-toml 把这一行注释绑定到 owner.name

# 下面列举 TOML 所支持的类型与格式要求
organization = "GitHub" # 字符串
bio = "GitHub Cofounder & CEO\nLikes tater tots and beer." # 字符串可以包含转义字符
dob = 1979-05-27T07:32:00Z # 日期, 使用 ISO 8601 Zulu 时区(最后的 Z 表示时区为 +00:00). 对 Go 来说兼容 RFC3339 layout.

[database]
server = "192.168.1.1"
ports = [ 8001, 8001, 8002 ] # 数组, 其元素类型也必须是TOML所支持的. Go 语言下类型是 slice
connection_max = 5000 # 整型, tom-toml 使用 int64 类型
enabled = true # 布尔型

[servers]

  # 可以使用缩进, tabs 或者 spaces 都可以, 毫无问题.
  [servers.alpha]
  ip = "10.0.0.1" # IP 格式只能用字符串了
  dc = "eqdc10"

  [servers.beta]
  ip = "10.0.0.2"
  dc = "eqdc10"

[clients]
data = [ ["gamma", "delta"], [1, 2] ] # 又一个数组
donate = 49.90 # 浮点, tom-toml 使用 float64 类型

# 通过 smtp 发电子邮件所需要的的参数
[smtpAuth]
Identity = ""
Username = "Do_Not_Reply"
Password = "password"
Host     = "example.com"
Subject  = "message"
To       = ["me@example.com","you@example.com"]
```

读取 `servers.alpha` 中的 ip 和 dc:

```go
package main

import (
	"fmt"
	"github.com/achun/tom-toml"
)

type smtpAuth struct {
	Identity string
	Username string
	Password string
	Host     string
	Subject  string
	To       []string
}

func main() {
	conf, err := toml.LoadFile("good.toml")

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(conf["servers.alpha.ip"].String())
	fmt.Println(conf["servers.alpha.dc"].String())

	sa := smtpAuth{}
	auth := conf.Fetch("smtpAuth") // 严格区分大小写

	sa.To = make([]string, auth["To"].Len())
	auth.Apply(&sa)

	fmt.Println(sa)
}
```

输出是这样的:

	10.0.0.1
	eqdc10
	{ Do_Not_Reply password example.com message [me@example.com you@example.com]}

您应该注意到了注释的表现形式, tom-toml 提供了注释支持.

## 注意

先写下解释用的 TOML 文本
```toml
[nameOftable]     # Kind() 为 TableName, String() 同此行
key1 = "v1"       # Kind() 为 String, String() 是 "v1"
key2 = "v2"       # Kind() 为 String, String() 是 "v2"
[[arrayOftables]] # Kind() 为 ArrayOfTables, String() 是此行及以下行
key3 = "v3"       # Kind() 为 String, String() 是 "v3"
```

因为采用 `map` 和支持注释的原因, 使用上有些特别. Toml 对象中存储的

 - TableName 仅是 TOML 规范中的 `[nameOftable]` 的字面值.
 - Table 仅是 TOML 规范中的 `[arrayOftables]` 的一个 Table.

因此用 `tm` 表示上述 Toml 对象的话

    tm["nameOftable"]       仅仅是 `[nameOftable]`, 不包含 Key/Value 部分
    tm["arrayOftables"]     是全部的 `arrayOftables`, 因为它是数组
    tm.Fetch("nameOftable") 是`[nameOftable]`的 Key/Value 部分, 类型是 Toml
    tm["nameOftable.key1"]  直接访问到了值为 "v1" 的数据
    t:=tm["arrayOftables"].Table(0) 是第一个 Table, Kind() 是 TableBody
    t["key3"] Key3          只能这样访问到


可以看出

 - 只有通过 `Fetch()` 方法才能得到一个 TOML 规范中定义的 Table 的主体.
 - 只有通过 `Table()` 方法才能得到 `Table` 类型.
 - `arrayOftables.key3` 这种写法是错误的, 不满足 TOML 规范的定义

map 带来 “nameOftable.key1” 这种点字符串方便的同时也产生了一些不便.
map 进行存储的话只能是这样, 就算不支持注释, 也逃不过 ArrayOfTables 的古怪.


## 贡献

请使用 GitHub 系统提出 issues 或者 pull 补丁到
[achun/tom-toml](https://github.com/achun/tom-toml). 欢迎任何反馈！


## License
Copyright (c) 2014, achun
All rights reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice, this
  list of conditions and the following disclaimer in the documentation and/or
  other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
