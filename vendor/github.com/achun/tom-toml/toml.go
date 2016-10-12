package toml

import (
	"errors"
	"io/ioutil"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Toml 是一个 maps, 不是 tree 实现.
type Toml map[string]Item

// Must be use New() to got Toml, do not use Toml{}.
// 新建一个 Toml, 必须使用 New() 函数, 不要用 Toml{}.
// 因为 Toml 的实现需要有一个用于管理的 Id Value, New() 可以做到.
func New() Toml {
	tm := Toml{}
	tm[iD] = GenItem(0)
	return tm
}

/**
Id 返回用于管理的 ".ID..." 对象副本.
如果 Id 不存在, 会自动建立一个, 但这不能保证顺序的可靠性.
*/
func (tm Toml) Id() Value {
	id, ok := tm[iD]
	if !ok || id.idx <= 0 {
		id = GenItem(0)
		tm[iD] = id
	}
	return *id.Value
}

// String returns TOML layout string.
// 格式化输出带缩进的 TOML 格式.
func (p Toml) String() string {
	return p.string("", 0)
}

type kkId struct {
	kind Kind
	key  string // it is key of map
	id   int
}

type sortIdx []kkId

func (p sortIdx) Len() int      { return len(p) }
func (p sortIdx) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (p sortIdx) Less(i, j int) bool {
	return p[i].id < p[j].id
}

type sortKey []kkId

func (p sortKey) Len() int      { return len(p) }
func (p sortKey) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (p sortKey) Less(i, j int) bool {
	return p[i].key < p[j].key
}

// prefix for nested TOML
func (p Toml) string(prefix string, indent int) (fmt string) {
	l := len(p)
	if l == 1 {
		return
	}

	// outputs end-of-line comments for ArrayOfTables
	// 输出嵌套TOML的行尾注释, 前一个 Toml 负责输出 ArrayOfTabes 的 Key
	id := p.Id()
	if id.idx <= 0 {
		return
	}

	if prefix != "" {
		prefix = prefix + "."
	}

	indentstr := strings.Repeat("\t", indent)

	// 如果有 prefix 那一定是嵌套的.
	if prefix == "" && id.eolComment != "" {
		fmt += " " + id.eolComment + "\n"
	}

	// 收集整理 kind,Key,idx 信息, 以便有序输出.
	var tops sortIdx
	var tabs sortIdx
	var vals sortKey

	for rawkey, it := range p {

		// ???需要更严格的检查
		key := strings.TrimSpace(rawkey)
		if key == "" || key != rawkey || !it.IsValid() {
			continue
		}

		ki := kkId{
			it.kind,
			key,
			it.idx,
		}

		pos := strings.LastIndex(key, ".")

		if ki.kind < TableName && pos == -1 {
			// Top level Key-Vlaue
			tops = append(tops, ki)
			continue
		}

		if ki.kind < TableName {
			// Key-Value
			vals = append(vals, ki)
		} else {
			tabs = append(tabs, ki)
			// TableName and ArrayOfTables
		}
	}

	sort.Sort(tops)
	sort.Sort(tabs)
	sort.Sort(vals)

	// Top level Key-Vlaue
	for _, kv := range tops {
		it := p[kv.key]

		for _, s := range it.multiComments {
			fmt += indentstr + s + "\n"
		}
		fmt += indentstr + kv.key + " = " + it.string(indentstr, 1)

		if it.eolComment != "" {
			fmt += " " + it.eolComment + "\n"
		}

	}

	if len(tops) != 0 {
		fmt += "\n"
	}

	// TableName and ArrayOfTables
	kvindent := indentstr + "\t"
	for _, kv := range tabs {

		it := p[kv.key]

		// ArrayOfTables
		if it.kind == ArrayOfTables {
			fmt += "\n"
			nested := kv.key
			if prefix != "" {
				nested = prefix + nested
			}
			for _, tm := range it.TomlArray() {
				id := tm.Id()

				for _, s := range id.multiComments {
					fmt += indentstr + s + "\n"
				}

				if id.eolComment == "" {
					fmt += indentstr + "[[" + prefix + kv.key + "]]\n"
				} else {
					fmt += indentstr + "[[" + prefix + kv.key + "]] " + id.eolComment + "\n"
				}

				fmt += tm.string(nested, indent+1)
			}
			continue
		}

		// TableName
		if fmt != "" {
			fmt += "\n"
		}

		for _, s := range it.multiComments {
			fmt += indentstr + s + "\n"
		}

		if it.eolComment == "" {
			fmt += indentstr + "[" + prefix + kv.key + "]\n"
		} else {
			fmt += indentstr + "[" + prefix + kv.key + "] " + it.eolComment + "\n"
		}

		// Key-Value
		tableName := kv.key + "."

		for _, kv := range vals {
			if !strings.HasPrefix(kv.key, tableName) {
				continue
			}

			key := kv.key[len(tableName):]

			// key 有 ".", 那么后续再输出
			if strings.Index(key, ".") != -1 {
				continue
			}

			it := p[kv.key]

			if len(it.multiComments) != 0 {
				fmt += "\n"
				for _, s := range it.multiComments {
					fmt += kvindent + s + "\n"
				}
			}

			if it.eolComment == "" {
				fmt += kvindent + key + " = " + it.string(kvindent, 1) + "\n"
			} else {
				fmt += kvindent + key + " = " + it.string(kvindent, 1) + " " +
					it.eolComment + "\n"
			}
		}
	}

	// TOML 最后的多行注释
	if prefix == "" {
		for _, s := range id.multiComments {
			fmt += indentstr + s + "\n"
		}
	}
	return
}

// Fetch returns Sub Toml of p, and reset name. not clone.

/**
such as:
	p.Fetch("")       // returns all valid elements in p
	p.Fetch("prefix") // same as p.Fetch("prefix.")
从 Toml 中提取出 prefix 开头的所有 Table 元素, 返回值也是一个 Toml.
注意:
	返回值是原 Toml 的子集.
	返回子集中不包括 [prefix] TableName.
	对返回子集添加 *Item 不会增加到原 Toml 中.
	对返回子集中的 *Item 进行更新, 原 Toml 也会更新.
	子集中不会含有 ArrayOfTables 类型数据.
*/
func (p Toml) Fetch(prefix string) Toml {
	nt := Toml{}
	ln := len(prefix)
	if ln != 0 {
		if prefix[ln-1] != '.' {
			prefix += "."
			ln++
		}
	}

	for key, it := range p {
		if !it.IsValid() || strings.Index(key, prefix) != 0 {
			continue
		}
		newkey := key[ln:]
		if newkey == "" {
			continue
		}
		nt[newkey] = it
	}
	return nt
}

// TableNames returns all name of TableName and ArrayOfTables.
// 返回所有 TableName 的名字和 ArrayOfTables 的名字.
func (p Toml) TableNames() (tableNames []string, arrayOfTablesNames []string) {
	for key, it := range p {
		if it.IsValid() {
			if it.kind == TableName {
				tableNames = append(tableNames, key)
			} else if it.kind == ArrayOfTables {
				arrayOfTablesNames = append(arrayOfTablesNames, key)
			}
		}
	}
	return
}

// Apply to each field in the struct, case sensitive.
/**
Apply 把 p 存储的值赋给 dst , TypeOf(dst).Kind() 为 reflect.Struct, 返回赋值成功的次数.
*/
func (p Toml) Apply(dst interface{}) (count int) {
	var (
		vv reflect.Value
		ok bool
	)

	vv, ok = dst.(reflect.Value)
	if ok {
		vv = reflect.Indirect(vv)
	} else {
		vv = reflect.Indirect(reflect.ValueOf(dst))
	}
	return p.apply(vv)
}

func (p Toml) apply(vv reflect.Value) (count int) {

	var it Item
	vt := vv.Type()
	if !vv.IsValid() || !vv.CanSet() || vt.Kind() != reflect.Struct || vt.String() == "time.Time" {
		return
	}

	for i := 0; i < vv.NumField(); i++ {
		name := vt.Field(i).Name
		it = p[name]

		if !it.IsValid() {
			continue
		}

		if it.kind == TableName {
			count += p.Fetch(name).Apply(vv.Field(i))
		} else {
			count += it.apply(vv.Field(i))
		}
	}
	return
}

var (
	InValidFormat = errors.New("invalid TOML format")
	Redeclared    = errors.New("duplicate definitionin")
)

// 从 TOML 格式 source 解析出 Toml 对象.
func Parse(source []byte) (tm Toml, err error) {
	p := &parse{Scanner: NewScanner(source)}

	tb := newBuilder(nil)

	p.Handler(
		func(token Token, str string) error {
			tb, err = tb.Token(token, str)
			return err
		})

	p.Run()
	tm = tb.root.Toml()
	tm[iD].multiComments = tb.comments
	return
}

// 如果 p!=nil 表示是子集模式, tablename 必须有相同的 prefix
type tomlBuilder struct {
	tm        Toml
	root      *tomlBuilder
	p         *tomlBuilder
	it        *Item
	iv        *Value
	comments  aString // comment or comments
	tableName string  // cache tableName
	prefix    string  // with "." for nested TOML
	token     Token   // 有些时候需要知道上一个 token, 比如尾注释
}

func newBuilder(root *tomlBuilder) tomlBuilder {
	tb := tomlBuilder{}

	tb.tm = New()

	if root == nil {
		tb.root = &tb
	} else {
		tb.root = root
		tb.token = tb.root.token
	}
	return tb
}

func (t tomlBuilder) Toml() Toml {
	return t.tm
}

func (t tomlBuilder) Token(token Token, str string) (tomlBuilder, error) {
	defer func() {
		// 缓存上一个 token, eolComment 等需要用
		if token == tokenWhitespace {
			return
		}

		t.root.token = token

		if token != tokenComment && token != tokenNewLine {
			t.token = token
		}
	}()
	switch token {
	case tokenError:
		return t.Error(str)
	case tokenRuneError:
		return t.RuneError(str)
	case tokenEOF:
		return t.EOF(str)
	case tokenWhitespace:
		return t.Whitespace(str)
	case tokenEqual:
		return t.Equal(str)
	case tokenNewLine:
		return t.NewLine(str)
	case tokenComment:
		return t.Comment(str)
	case tokenString:
		return t.String(str)
	case tokenInteger:
		return t.Integer(str)
	case tokenFloat:
		return t.Float(str)
	case tokenBoolean:
		return t.Boolean(str)
	case tokenDatetime:
		return t.Datetime(str)
	case tokenTableName:
		return t.TableName(str)
	case tokenArrayOfTables:
		return t.ArrayOfTables(str)
	case tokenKey:
		return t.Key(str)
	case tokenArrayLeftBrack: // [
		return t.ArrayLeftBrack(str)
	case tokenArrayRightBrack: // ]
		return t.ArrayRightBrack(str)
	case tokenComma:
		return t.Comma(str)
	}
	return t, NotSupported
}

func (t tomlBuilder) Error(str string) (tomlBuilder, error) {
	return t, errors.New(str)
}

func (t tomlBuilder) RuneError(str string) (tomlBuilder, error) {
	return t, errors.New(str)
}

func (t tomlBuilder) EOF(str string) (tomlBuilder, error) {
	return t, nil
}

func (t tomlBuilder) Whitespace(str string) (tomlBuilder, error) {
	return t, nil
}

func (t tomlBuilder) NewLine(str string) (tomlBuilder, error) {
	return t, nil
}

func (t tomlBuilder) Comment(str string) (tomlBuilder, error) {

	// eolComment
	if t.root.token != tokenEOF && t.root.token != tokenNewLine {

		if len(t.comments) != 0 {
			return t, InternalError
		}

		// [[aot]] #comment, save to iD.eolComment
		if t.root.token == tokenArrayOfTables {
			id, ok := t.tm[iD]
			if !ok || id.eolComment != "" {
				return t, InternalError
			}
			id.Value.eolComment = str
			return t, nil
		}

		if t.iv == nil && t.it == nil {
			return t, InternalError
		}

		if t.iv == nil {
			if t.it.eolComment == "" {
				t.it.eolComment, t.comments = str, aString{}
			}
		} else if t.iv.eolComment == "" {
			t.iv.eolComment, t.comments = str, aString{}
		} else {
			return t, InternalError
		}

		return t, nil
	}

	// multiComments
	t.comments = append(t.comments, str)
	return t, nil
}

func (t tomlBuilder) String(str string) (tomlBuilder, error) {
	if t.iv == nil {
		return t, InternalError
	}

	str, err := strconv.Unquote(str)
	if err != nil {
		return t, err
	}

	if t.iv.kind != Array && t.iv.kind != StringArray {
		return t, t.iv.SetAs(str, String)
	}
	return t, t.iv.Add(str)
}

func (t tomlBuilder) Integer(str string) (tomlBuilder, error) {
	if t.iv == nil {
		return t, InternalError
	}

	if t.iv.kind != Array && t.iv.kind != IntegerArray {
		return t, t.iv.SetAs(str, Integer)
	}
	v, err := conv(str, Integer)
	if err != nil {
		return t, err
	}
	return t, t.iv.Add(v)
}
func (t tomlBuilder) Float(str string) (tomlBuilder, error) {
	if t.iv == nil {
		return t, InternalError
	}

	if t.iv.kind != Array && t.iv.kind != FloatArray {
		return t, t.iv.SetAs(str, Float)
	}
	v, err := conv(str, Float)
	if err != nil {
		return t, err
	}
	return t, t.iv.Add(v)
}
func (t tomlBuilder) Boolean(str string) (tomlBuilder, error) {
	if t.iv == nil {
		return t, InternalError
	}

	if t.iv.kind != Array && t.iv.kind != BooleanArray {
		return t, t.iv.SetAs(str, Boolean)
	}
	v, err := conv(str, Boolean)
	if err != nil {
		return t, err
	}
	return t, t.iv.Add(v)
}
func (t tomlBuilder) Datetime(str string) (tomlBuilder, error) {
	if t.iv == nil {
		return t, InternalError
	}

	if t.iv.kind != Array && t.iv.kind != DatetimeArray {
		return t, t.iv.SetAs(str, Datetime)
	}
	v, err := conv(str, Datetime)
	if err != nil {
		return t, err
	}
	return t, t.iv.Add(v)
}

func (t tomlBuilder) TableName(str string) (tomlBuilder, error) {
	path := str[1 : len(str)-1]

	it, ok := t.tm[path]
	if ok {
		return t, Redeclared
	}

	comments := t.comments
	t.comments = aString{}

	if t.prefix != "" {
		if t.p == nil {
			return t, InternalError
		}

		if path == t.prefix {
			return t, Redeclared
		}

		if !strings.HasPrefix(path, t.prefix+".") {
			t = *t.p
			t.comments = comments
			return t.TableName(str)
		}
		path = path[len(t.prefix)+1:]
	}

	// cached tableName for Key
	t.tableName = path

	it = GenItem(TableName)

	it.multiComments = append(it.multiComments, comments...)

	t.tm[path] = it
	t.it = &it
	t.iv = nil
	return t, nil
}

func (t tomlBuilder) Key(str string) (tomlBuilder, error) {

	it := GenItem(0)

	it.multiComments, t.comments = t.comments, aString{}

	if t.tableName != "" {
		str = t.tableName + "." + str
	}

	t.tm[str] = it
	t.iv = it.Value

	return t, nil
}

func (t tomlBuilder) Equal(str string) (tomlBuilder, error) {

	if t.root.token != tokenKey {
		return t, InValidFormat
	}

	if t.iv == nil {
		return t, InternalError
	}
	return t, nil
}

func (t tomlBuilder) ArrayOfTables(str string) (nt tomlBuilder, err error) {
	path := str[2 : len(str)-2]

	if t.prefix != "" {
		if t.p == nil {
			return t, InternalError
		}

		comments := t.comments

		// 增加兄弟 table
		if t.prefix == path {
			t = *t.p
			t.comments = comments
			return t.ArrayOfTables(str)
		} else if !strings.HasPrefix(path, t.prefix+".") {
			// 递归向上
			t = *t.p
			t.comments = comments
			return t.ArrayOfTables(str)
		}
		path = path[len(t.prefix)+1:]
	}

	return t.nestToml(path)
}

// 嵌套 TOML , prefix 就是 [[arrayOftablesName]]
func (t tomlBuilder) nestToml(prefix string) (tomlBuilder, error) {

	it, ok := t.tm[prefix]

	// [[foo.bar]] 合法性检查不够完全??????
	if ok && it.kind != ArrayOfTables {
		return t, Redeclared
	}

	tb := newBuilder(t.root)

	tb.p = &t
	tb.tm = New()

	id := tb.tm[iD]

	// Comments
	id.multiComments, t.comments = t.comments, aString{}

	// first [[...]]
	if !ok {
		it = GenItem(ArrayOfTables)
		it.v = TomlArray{tb.tm}
		t.tm[prefix] = it

	} else {
		// again [[...]]
		ts := it.v.(TomlArray)
		it.v = append(ts, tb.tm)
	}
	tb.prefix = prefix
	return tb, nil
}

func (t tomlBuilder) ArrayLeftBrack(str string) (tomlBuilder, error) {
	if t.iv == nil {
		return t, NotSupported
	}

	if t.iv.kind == InvalidKind {
		t.iv.kind = Array
		return t, nil
	}
	if t.iv.kind != Array {
		return t, NotSupported
	}

	nt := t
	nt.iv = NewValue(Array)
	nt.p = &t
	t.iv.Add(nt.iv)
	return nt, nil
}

func (t tomlBuilder) ArrayRightBrack(str string) (tomlBuilder, error) {

	if t.iv == nil || t.iv.kind < StringArray || t.iv.kind > Array {
		return t, InValidFormat
	}

	if t.p == nil {
		return t, nil
	}
	return *t.p, nil
}

func (t tomlBuilder) Comma(str string) (tomlBuilder, error) {
	if t.iv == nil || t.iv.kind < StringArray || t.iv.kind > Array {
		return t, InValidFormat
	}

	return t, nil
}

// Create a Toml from a file.
// 便捷方法, 从 TOML 文件解析出 Toml 对象.
func LoadFile(path string) (toml Toml, err error) {
	source, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	toml, err = Parse(source)
	return
}
