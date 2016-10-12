# 问题

这里讨论基于用 map 实现 TOML 的问题.

当然, 我们把访问路径用 "." 连接. 这样的 map 访问数据非常直接方便. 你可以用

```go
tm["k.e.y"]  // 一下就访问到最终的目标
// 而不用像这样
tm.Get("k").Get("e").Get("y")
tm.Get("k.e.y")
```

TOML v0.2.0 的定义中, Table/ArrayOfTables 是可以深层嵌套的.

参见 https://github.com/mojombo/toml/pull/153

用 map 实现 TOML, 最简单的方法是:

```go
map[string]interface{}
```

但是这种万能接口, 使用者还需要 type assertion, 并不方便, 所以采用 struct 是需要的.

# 分析

先看看 TOML 都定义了什么?

 - Comment       注释. 前置(多行)注释和行尾注释还有 TOML 文本最后的注释
 - Value         String, Integer, Float, Boolean, Datetime 以及数组形式
 - Key           给 Value 命名, 以便访问
 - Table         一组 Key = Value 的集合
 - TableName     给 Table 命名
 - ArrayOfTables 正如其名, 可以看作是 嵌套TOML
 - TablesName    给 ArrayOfTables 命名, 它表示了一组 Table 的名字

TOML 文本最后的注释比较特别, 只有一个. 可以特殊处理, 比如用 key="", 下文用 TomlComments 表示.

显而易见的规则:

    非 ArrayOfTables 中的 Value 是可以直接 key 访问.

    TableName 也是可以直接 "k.e.y" 访问. TableName 成了 Value 的一种类型.

    tm["k.e.y"] 里面的 key 其实是个访问路径, 由 "TableName[.Key]" 组成. 这是个完整的全路径.

    Table 中可以嵌套 Table, 同样用 "k.e.y" 访问.

    ArrayOfTables 中可以嵌套 ArrayOfTables, 下文分析访问方式

    Table, ArrayOfTables 可以互相嵌套.

    在 map 中无法直接用 tm["k.e.y"] 访问到 ArrayOfTables 中的元素, 因为 key 中没法让数组下标生效, 如果加入下标,维护将会很麻烦.
    正如笔者前面说的, 如果把 ArraOfTables 当作 `Array Of TOML` 这就很容易理解了. 
    还不如直接命名为 TOMLArray 来的简单明了.

    注释. ArrayOfTables 中的每一个下标 [[TablesName]] 也允许有注释.

    格式化输出 TOML文本要求数据必须能被有序访问.

对 Toml 定义的影响:

```go
type Toml map[string]Item
```

Item 有可能是

 - Value
 - TableName
 - ArrayOfTables 经过前述分析就是 嵌套TOML.
 - 内部实现需要的 "." 开头的数据

Value 的 Kind 包括

    String
    Integer
    Float
    Boolean
    Datetime
    StringArray
    IntegerArray
    FloatArray
    BooleanArray
    DatetimeArray
    Array         元素是 xxxxArray 类型, 规范没有明确是否可以 Array 嵌套 Array.

TableName 和 ArrayOfTables 是独立的, 就是他们自己.

实现的时候, 当然所有的数据都用 interface{} 保存在 Value 结构中. 只不过在接口上 Value和Item有所区别. 实际上

    Value 的接口囊括了 TableName 的支持.
    ArrayOfTables 只有在 Item 接口中才能访问到.


# 结果

定义

```go
type Value struct{}  // 省略细节, 包含 TableName 类型

type Toml map[string]Item

func (t Toml) Fetch(prefix string) Toml // 返回 "prefix." 开头的子集, 并省略它

type Tables []Toml  // 官方没有给出具体名字, 只有造一个

type Item struct{
    *Value
}

func (i Item) TomlArray() Tables // 如果 kind 是ArrayOfTables 的话

```

Item 导出 Value 可以方便一些操作, 目前 Item 只是多支持了 ArrayOfTables.
尽管有些不同, Kind 的命名尽量采用 TOML 定义的字面值.

# 访问

前面分析过"k.e.y"是个完全路径, 总是写完全路径有时候不是很方便. tom-toml 提供便捷方法  `Fetch(prefix)` 简化访问. 所以, 访问路径有两种情况.

## 完全路径

形式为
 - "Key"           访问顶层 Key-Value, 它不在 Table 中
 - "TableName"     访问 TableName 本身, 不包含 "Key = Value" 部分
 - "TableName.Key" 访问 Table 中的 Key-Value
 - "ArrayOfTables" 得到 Toml 数组, 访问某个 Toml 中的元素仍然用完全路径

即: 用 "." 连接的 TableName 和 Key 路径.

*吐槽:事儿本来很容易理解, 用文字表达清晰却不容易, 还容易发生歧义. 只是 map 的 key而已, 地球人都知道.*

举例: 注释中直接写上对应访问代码, tm 代表对应的 Toml 对象.

```toml
id = 1                            # tm["id"]
[user]                            # tm["user"] // returns TableName only
    name = "Jack"                 # tm["user.name"]
[user.profile]                    # tm["user.profile"] // TableName only
    email = "jack@exampl.com"     # tm["user.profile.email"]

[[users]]                         # t := tm["users"][0]
    id = 1                        # t["id"]
    [users.jack]                  # t["jack"]
        email = "jack@exampl.com" # t["jack.email"]

[[users]]                         # t := tm["users"][1]
    id = 2                        # t["id"]
    [users.tom]                   # t["tom"]
        email = "tom@exampl.com"  # t["tom.email"]
    [[users.tom.follows]]         # ft := t["tom.follows"][0]
        name = "john"             # ft["name"]
```

访问 ArrayOfTables 中的Key-Value, 看上去更像是用了相对路径, 解释见下文.

*吐槽:不必太在意把 ArrayOfTables 也算做全路径访问, 这样做对简化代码实现和操作很有效*

## 相对路径 

相对路径由方法 Fetch 方法支持. 对应上面的例子, 使用
```go
user := tm.Fetch("user")
```

那么得到的 user 和访问是这样的

```toml
name = "Jack"                     # user["name"]
[profile]                         # user["profile"] // TableName only
    email = "jack@exampl.com"     # user["profile.email"]
```

如果使用 
```go
profile := tm.Fetch("user.profile")
```

那么得到的 user 和访问是这样的

```toml
email = "jack@exampl.com"     # profile["email"]
```

当然, 这个 "prefix" 必须是个 TableName. 否则无法工作, 只能得到空集.

```go
bad := Fetch("user.name") // 无法工作
```

Fetch 也无法作用于 ArrayTables , 下面的代码也无法工作, 只能得到空集.

```go
users := tm.Fetch("users")
```

下面的代码解释无法工作的原因

```toml
[[users]]                         # 如果可以, users 将被减省
    id = 1                        # id = 1
    [users.jack]                  # [jack]
        email = "jack@exampl.com" #     email = "jack@exampl.com"
                                   
[[users]]                         # 下面的数据就无法安排了
    id = 2                        # id = 2 # Invalid, 和上面的 id 重复了
    [users.tom]                   #
        email = "tom@exampl.com"  #
    [[users.tom.follows]]         #
        name = "john"             #
```

原因很简单, ArrayOfTables 是数组, Key 会发生重复, 合并数组会产生数据覆盖.

## 嵌套TOML

通过上述分析, ArrayOfTables 也可以看作向 dst TOML 中嵌入了多个 Src1,Src2,Src3 ... TOML, 也就是 嵌套TOML.
可以看作为了保障 key 不重复加了个不重复的前缀. 这给合并多个 TOML 文档提供了可能.

# 限制

 - 保留元素     保留 key 以"."开头的元素. 修改这些元素会产生不可预计的结果.
 - 禁止循环嵌套 因可以用函数生成Toml, 循环嵌套有可能发生.
 
限制的原因和内部实现有关, 不细述.

# 疑问

有一些未确定的问题

## ArrayOfTables

官方申明下面 Table 的文档是合法的.

```toml
# [x] you
# [x.y] don't
# [x.y.z] need these
[x.y.z.w] # for this to work
```

官方申明下面 ArrayOfTables的文档是非法的.

```toml
# INVALID TOML DOC
[[fruit]]
  name = "apple"

  [[fruit.variety]]
    name = "red delicious"

  # This table conflicts with the previous table
  [fruit.variety]
    name = "granny smith"
```

官方文档未明确下面的文档是否合法.

没有声明 `[foo]` 或者 `[[foo]]`, 直接

```toml
[[foo.bar]]
```

这应该是非法的, 因为如果补全这种写法的话, 可能是

```toml
[foo]
[[foo.bar]]
```

也有可能是

```toml
[[foo]]
[[foo.bar]]
```

会产生歧义.

