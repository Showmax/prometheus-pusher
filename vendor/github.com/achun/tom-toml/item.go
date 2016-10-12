package toml

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Kind 用来标识 TOML 所有的规格.
// 对于用于配置的 TOML 定义, Kind 其实是分段定义和值定义的.
// 由于 TOML 官方没有使用 段/值 这样的词汇, tom-toml 选用规格这个词.
type Kind uint

// don't change order
const (
	InvalidKind Kind = iota
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
	Array
	// TableName 因为 tom-toml 支持注释的原因, 不存储具体值数据, 只存储规格本身信息.
	// 又因为 Toml 是个 map, 本身就具有 key/value 的存储功能, 所以无需另外定义 Table
	TableName
	ArrayOfTables
)

// 内部使用的 id
var iD = time.Now().UTC().Format(".ID20060102150405.000000000")

func (k Kind) String() string {
	return kindsName[k]
}

var kindsName = [...]string{
	"InvalidKind",
	"String",
	"Integer",
	"Float",
	"Boolean",
	"Datetime",
	"StringArray",
	"IntegerArray",
	"FloatArray",
	"BooleanArray",
	"DatetimeArray",
	"Array",
	"TableName",
	"ArrayOfTables",
}

var (
	NotSupported  = errors.New("not supported")
	OutOfRange    = errors.New("out of range")
	InternalError = errors.New("internal error")
	InvalidItem   = errors.New("invalid Item")
)

// 计数器为保持格式化输出次序准备.
var _counter = 0
var _counterLocker sync.Mutex

func counter(idx int) int {
	if idx > 0 {
		return idx
	}
	_counterLocker.Lock()
	defer _counterLocker.Unlock()
	_counter++
	return _counter
}

// GenItem 函数返回一个新 Item.
// 为保持格式化输出次序, GenItem 内部使用了一个计数器.
// 使用者应该使用该函数来得到新的 Item. 而不是用 new(Item) 获得.
// 那样的话就无法保持格式化输出次序.
func GenItem(kind Kind) Item {
	if kind < 0 || kind > ArrayOfTables {
		panic(NotSupported)
	}

	it := Item{}
	it.Value = &Value{}
	it.multiComments = []string{}
	it.kind = kind

	it.idx = counter(0)

	if kind == ArrayOfTables {
		it.v = TomlArray{}
	}

	return it
}

type aString []string

func (s aString) String() string {
	return strings.Join(s, "\n")
}

// Value 用来存储 String 至 Array 的值.
type Value struct {
	kind          Kind
	idx           int
	v             interface{}
	eolComment    string  // end of line comment
	multiComments aString // Multi-line comments
	//key           string  // cached key name for TOML formatter
}

// NewValue 函数返回一个新 *Value.
// 为保持格式化输出次序, NewValue 内部使用了一个计数器.
// 使用者应该使用该函数来得到新的 *Value. 而不是用 new(Value) 获得.
// 那样的话就无法保持格式化输出次序.
func NewValue(kind Kind) *Value {
	if kind > TableName {
		return nil
	}

	it := &Value{}
	it.kind = kind
	it.multiComments = []string{}

	it.idx = counter(0)

	return it
}

// returns multiline comments
func (p *Value) Comments() []string {
	return append([]string{}, p.multiComments...)
}

// returns end-of-line comment
func (p *Value) Comment() string {
	return p.eolComment
}

// set multiline comments
func (p *Value) SetComments(as []string) {
	c := []string{}
	for _, s := range as {
		if strings.IndexAny(s, "\n\r\x1E") >= 0 {
			return
		}

		s = strings.TrimSpace(s)
		if s != "" && s[0] != '#' {
			s = "# " + s
		}
		c = append(c, s)
	}
	p.multiComments = c
	return
}

// set end-of-line comment
func (p *Value) SetComment(s string) {
	if strings.IndexAny(s, "\n\r\x1E") >= 0 {
		return
	}

	s = strings.TrimSpace(s)
	if s != "" && s[0] != '#' {
		s = "# " + s
	}

	p.eolComment = s
}

// Kind 返回数据的风格.
func (p *Value) Kind() Kind {
	if !p.IsValid() {
		return InvalidKind
	}
	return p.kind
}

/**
Id 返回 int 值, 此值表示 Value 在运行期中生成次序的唯一序号.
返回 0 表示该 Value 无效.
*/
func (p *Value) Id() int {
	if !p.IsValid() {
		return 0
	}
	return p.idx
}

func (p *Value) KindIs(kind ...Kind) bool {
	if p == nil {
		return false
	}
	for _, k := range kind {
		if p.kind == k {
			return true
		}
	}
	return false
}

// IsValid 返回 p 是否有效.
func (p *Value) IsValid() bool {
	return p != nil && p.kind != InvalidKind && (p.v != nil || p.kind == TableName)
}

// IsValue 返回 p 是否存储了值数据
func (p *Value) IsValue() bool {
	return p != nil && p.kind != InvalidKind && p.v != nil && p.kind < TableName
}

func (p *Value) canNotSet(k Kind) bool {
	return p == nil || p.kind != InvalidKind && p.kind != k
}

// Set 用来设置 *Value 要存储的具体值. 参数 x 的类型范围可以是
// String,Integer,Float,Boolean,Datetime 之一
// 如果 *Value 的 Kind 是 InvalidKind(也就是没有明确值类型),
// 调用 Set 后, *Value 的 kind 会相应的更改, 否则要求 x 的类型必须符合 *Value 的 kind
// Set 失败会返回 NotSupported 错误.
func (p *Value) Set(x interface{}) error {
	if p == nil {
		return NotSupported
	}
	switch v := x.(type) {
	case string:
		if p.canNotSet(String) {
			return NotSupported
		}
		p.v = v
		p.kind = String
	case bool:
		if p.canNotSet(Boolean) {
			return NotSupported
		}
		p.v = v
		p.kind = Boolean
	case float64:
		if p.canNotSet(Float) {
			return NotSupported
		}
		p.v = v
		p.kind = Float
	case time.Time:
		if p.canNotSet(Datetime) {
			return NotSupported
		}
		p.v = v.UTC()
		p.kind = Datetime
	case int64:
		if p.canNotSet(Integer) {
			return NotSupported
		}
		p.v = int64(v)
		p.kind = Integer
	case int:
		if p.canNotSet(Integer) {
			return NotSupported
		}
		p.v = int64(v)
		p.kind = Integer
	case uint:
		if p.canNotSet(Integer) {
			return NotSupported
		}
		p.v = int64(v)
		p.kind = Integer
	case int8:
		if p.canNotSet(Integer) {
			return NotSupported
		}
		p.v = int64(v)
		p.kind = Integer
	case int16:
		if p.canNotSet(Integer) {
			return NotSupported
		}
		p.v = int64(v)
		p.kind = Integer
	case int32:
		if p.canNotSet(Integer) {
			return NotSupported
		}
		p.v = int64(v)
		p.kind = Integer
	case uint8:
		if p.canNotSet(Integer) {
			return NotSupported
		}
		p.v = int64(v)
		p.kind = Integer
	case uint16:
		if p.canNotSet(Integer) {
			return NotSupported
		}
		p.v = int64(v)
		p.kind = Integer
	case uint32:
		if p.canNotSet(Integer) {
			return NotSupported
		}
		p.v = int64(v)
		p.kind = Integer
	case uint64:
		if p.canNotSet(Integer) {
			return NotSupported
		}
		if v > 9223372036854775807 {
			return OutOfRange
		}
		p.kind = Integer
		p.v = int64(v)
	default:
		return NotSupported
	}
	p.idx = counter(p.idx)
	return nil
}

func conv(s string, kind Kind) (v interface{}, err error) {
	switch kind {
	case String:
		v = s
	case Integer:
		v, err = strconv.ParseInt(s, 10, 64)
	case Float:
		v, err = strconv.ParseFloat(s, 64)
	case Boolean:
		v, err = strconv.ParseBool(s)
	case Datetime:
		v, err = time.Parse(time.RFC3339, s) // time zone +00:00
		if err == nil {
			v = v.(time.Time).UTC()
		}
	default:
		err = NotSupported
	}
	return
}

// SetAs是个便捷方法, 通过参数 kind 对 string 参数进行转换并执行 Set.
func (p *Value) SetAs(s string, kind Kind) (err error) {
	if p.canNotSet(kind) {
		return NotSupported
	}
	switch kind {
	case String:
		p.v = s
	case Integer:
		var v int64
		v, err = strconv.ParseInt(s, 10, 64)
		if err == nil {
			p.v = v
		}
	case Float:
		var v float64
		v, err = strconv.ParseFloat(s, 64)
		if err == nil {
			p.v = v
		}
	case Boolean:
		var v bool
		v, err = strconv.ParseBool(s)
		if err == nil {
			p.v = v
		}
	case Datetime:
		var v time.Time
		v, err = time.Parse(time.RFC3339, s)
		if err == nil {
			p.v = v.UTC()
		}
	default:
		return NotSupported
	}
	if err == nil {
		p.kind = kind
		p.idx = counter(p.idx)
	}
	return
}

func asValue(i interface{}) (v *Value, ok bool) {
	it, ok := i.(Item)
	if ok {
		v = it.Value
	} else {
		v, ok = i.(*Value)
	}
	return
}

// Add element for Array or typeArray.
/**
Add 方法为数组添加元素, 支持空数组元素.
p 本身的 Kind 决定是否支持参数元素 Kind.
*/
func (p *Value) Add(ai ...interface{}) error {
	if p == nil || p.kind < StringArray || p.kind > Array {
		return NotSupported
	}
	if len(ai) == 0 {
		return nil
	}

	vs := make([]*Value, len(ai))

	k := 0
	mix := false

	// 全部检查一遍
	for i, s := range ai {
		v, ok := asValue(s)

		if !ok {
			v = &Value{}
			err := v.Set(s)
			if err != nil {
				return err
			}
		}

		if v == nil || v.kind > Array {
			return NotSupported
		}

		if p.kind != Array && v.kind != p.kind+String-StringArray {
			return NotSupported
		}

		// 用于分析 kind 的情况
		k = k | 1<<v.kind
		if k != 1<<v.kind {
			mix = true
		}

		v.idx = counter(v.idx)
		vs[i] = v
	}

	if mix {
		nk := k >> (StringArray - 1) << (StringArray - 1)
		if p.kind != Array || nk != k {
			return NotSupported
		}
	} else {
		if p.kind == Array {
			if p.Len() > 0 && vs[0].kind < StringArray {
				return NotSupported
			}
			// plain
			if vs[0].kind < StringArray {
				p.kind = vs[0].kind + StringArray - String
			}
		}
	}

	if p.v == nil {
		p.v = make([]*Value, 0)
	}
	o, ok := p.v.([]*Value)

	if !ok {
		return NotSupported
	}

	p.v = append(o, vs...)
	return nil
}

// String 返回 *Value 存储数据的字符串表示.
// 注意所有的规格定义都是可以字符串化的.
func (p *Value) String() string {
	return p.string("", 0)
}

/**
如果值是 Integer 可以使用 Int 返回其 int64 值.
否则返回 0
*/
func (p *Value) Int() int64 {
	if !p.IsValid() || p.kind != Integer {
		return 0
	}
	return p.v.(int64)
}

/**
如果值是 Integer 可以使用 Integer 返回其 int 值.
否则返回 0
*/
func (p *Value) Integer() int {
	if !p.IsValid() || p.kind != Integer {
		return 0
	}
	return int(p.v.(int64))
}

/**
如果值是 Integer 可以使用 UInteger 返回其 uint 值.
否则返回 0. 注意此方法不检查值是否为负数.
*/
func (p *Value) UInteger() uint {
	if !p.IsValid() || p.kind != Integer {
		return 0
	}
	return uint(p.v.(int64))
}

/**
如果值是 Integer 可以使用 UInt 返回其 uint64 值.
否则返回 0
*/
func (p *Value) UInt() uint64 {
	if !p.IsValid() || p.kind != Integer {
		return 0
	}
	return uint64(p.v.(int64))
}

/**
如果值是 Float 可以使用 Float 返回其 float64 值.
否则返回 0
*/
func (p *Value) Float() float64 {
	if !p.IsValid() || p.kind != Float {
		return 0
	}
	return p.v.(float64)
}

/**
如果值是 Boolean 可以使用 Boolean 返回其 bool 值.
否则返回 false
*/
func (p *Value) Boolean() bool {
	if !p.IsValid() || p.kind != Boolean {
		return false
	}
	return p.v.(bool)
}

/**
如果值是 Datetime 可以使用 Datetime 返回其 time.Time 值.
否则返回UTC时间公元元年1月1日 00:00:00. 可以用 IsZero() 进行判断.
*/
func (p *Value) Datetime() time.Time {
	if !p.IsValid() || p.kind != Datetime {
		return time.Time{}
	}
	return p.v.(time.Time)
}

func (p *Value) StringArray() []string {
	if !p.IsValid() {
		return nil
	}
	a, ok := p.v.([]*Value)
	if !ok || p.kind != StringArray {
		return nil
	}
	re := make([]string, len(a))
	for i, v := range a {
		re[i] = v.String()
	}
	return re
}

func (p *Value) IntArray() []int64 {
	if !p.IsValid() {
		return nil
	}
	a, ok := p.v.([]*Value)
	if !ok || p.kind != IntegerArray {
		return nil
	}
	re := make([]int64, len(a))
	for i, v := range a {
		re[i] = v.Int()
	}
	return re
}

func (p *Value) UIntArray() []uint64 {
	if !p.IsValid() {
		return nil
	}
	a, ok := p.v.([]*Value)
	if !ok || p.kind != IntegerArray {
		return nil
	}
	re := make([]uint64, len(a))
	for i, v := range a {
		re[i] = v.UInt()
	}
	return re
}

func (p *Value) IntegerArray() []int {
	if !p.IsValid() {
		return nil
	}
	a, ok := p.v.([]*Value)
	if !ok || p.kind != IntegerArray {
		return nil
	}
	re := make([]int, len(a))
	for i, v := range a {
		re[i] = int(v.Int())
	}
	return re
}

func (p *Value) UIntegerArray() []uint {
	if !p.IsValid() {
		return nil
	}
	a, ok := p.v.([]*Value)
	if !ok || p.kind != IntegerArray {
		return nil
	}
	re := make([]uint, len(a))
	for i, v := range a {
		re[i] = uint(v.UInt())
	}
	return re
}

func (p *Value) FloatArray() []float64 {
	if !p.IsValid() {
		return nil
	}
	a, ok := p.v.([]*Value)
	if !ok || p.kind != FloatArray {
		return nil
	}
	re := make([]float64, len(a))
	for i, v := range a {
		re[i] = v.Float()
	}
	return re
}

func (p *Value) BooleanArray() []bool {
	if !p.IsValid() {
		return nil
	}
	a, ok := p.v.([]*Value)
	if !ok || p.kind != BooleanArray {
		return nil
	}
	re := make([]bool, len(a))
	for i, v := range a {
		re[i] = v.Boolean()
	}
	return re
}

func (p *Value) DatetimeArray() []time.Time {
	if !p.IsValid() {
		return nil
	}
	a, ok := p.v.([]*Value)
	if !ok || p.kind != DatetimeArray {
		return nil
	}
	re := make([]time.Time, len(a))
	for i, v := range a {
		re[i] = v.Datetime()
	}
	return re
}

// Len returns length for Array , typeArray.
// Otherwise Kind return -1.
// +dl

/**
Len 返回数组类型元素个数. 否则返回 -1.
*/
func (p *Value) Len() int {
	if p.IsValid() && p.kind >= StringArray && p.kind <= Array {
		a, ok := p.v.([]*Value)
		if ok {
			return len(a)
		}
	}
	return -1
}

// Index returns *Value for Array , typeArray.
// idx negative available.
// Otherwise Kind return nil.
// +dl

/**
Index 根据 idx 下标返回类型数组或二维数组对应的元素.
idx 可以用负数作为下标.
如果非数组或者下标超出范围返回 nil.
*/
func (p *Value) Index(idx int) *Value {
	if !p.IsValid() || p.kind < StringArray && p.kind > Array {
		return nil
	}
	a, ok := p.v.([]*Value)
	if !ok {
		return nil
	}
	size := len(a)
	if idx < 0 {
		idx = size + idx
	}
	if idx < 0 || idx >= size {
		return nil
	}
	return a[idx]
}

// for ArrayOfTables
type TomlArray []Toml

func (t TomlArray) Index(idx int) Toml {
	if idx < len(t) {
		return t[idx]
	}
	return nil
}

func (t TomlArray) Len() int {
	return len(t)
}

// Item 扩展自 *Value,支持 ArrayOfTables.
type Item struct {
	*Value
}

/**
如果是 ArrayOfTables 返回 TomlArray, 否则返回 nil.
使用返回的 TomlArray 时, 注意其数组特性.
*/
func (i Item) TomlArray() TomlArray {
	if i.Value == nil || i.Value.v == nil || i.kind != ArrayOfTables {
		return nil
	}
	t, ok := i.v.(TomlArray)
	if ok {
		return t
	}
	return nil
}

/**
如果是 ArrayOfTables 追加 toml, 返回发生的错误.
*/
func (i Item) AddTable(tm Toml) error {
	if i.Value == nil || i.Value.v == nil || i.kind != ArrayOfTables {
		return NotSupported
	}

	if len(tm) == 0 {
		return nil
	}

	if tm.Id().idx <= i.idx {
		return NotSupported
	}

	aot, ok := i.v.(TomlArray)
	if !ok {
		return InternalError
	}

	i.v = append(aot, tm)
	return nil
}

// Index returns Toml for ArrayOfTables[idx].
// Otherwise Kind return nil.
// +dl

/**
如果是 ArrayOfTables 返回下标为 idx 的 Toml, 否则返回 nil.
支持倒序下标.
*/
func (i Item) Table(idx int) Toml {
	if i.Value == nil || i.Value.v == nil || i.kind != ArrayOfTables {
		return nil
	}
	aot, ok := i.v.(TomlArray)
	if !ok {
		return nil
	}

	size := len(aot)
	if idx < 0 {
		idx = size + idx
	}
	if idx < 0 || idx >= size {
		return nil
	}
	return aot[idx]
}

// Len returns length for Array , typeArray , ArrayOfTables.
// Otherwise Kind return -1.
// +dl

/**
Len 返回数组类型的元素个数.
否则返回 -1.
*/
func (i Item) Len() int {
	if i.Value != nil && i.kind == ArrayOfTables {
		a, ok := i.v.(TomlArray)
		if ok {
			return len(a)
		}
		return -1
	}
	return i.Value.Len()
}

// Value 的 string, 只考虑 layout, 不考虑上层的缩进关系.
// indent 是专门为 typeArrayString 留的. 这样做可以简化代码.
func (p *Value) string(indent string, layout int) string {
	if !p.IsValid() {
		return ""
	}

	switch p.kind {
	case String:
		if layout == 0 {
			return p.v.(string)
		} else {
			return strconv.Quote(p.v.(string))
		}
	case Integer:
		return strconv.FormatInt(p.v.(int64), 10)
	case Float:
		return strconv.FormatFloat(p.v.(float64), 'f', -1, 64)
	case Boolean:
		return strconv.FormatBool(p.v.(bool))
	case Datetime:
		return p.v.(time.Time).Format("2006-01-02T15:04:05Z") // ISO 8601

	case StringArray, IntegerArray, FloatArray, BooleanArray, DatetimeArray:
		return p.typeArrayString(indent, 1)
	case Array:
		return p.typeArrayString(indent, 1)
		/*
			case TableName:
				return "[]"
			case ArrayOfTables:
				return "[[" + p.key + "]]"
			default:
				return InvalidItem.Error()
		*/
	}
	return ""
}

// typeArray 比较特殊, 单独处理缩进问题
func (p *Value) typeArrayString(indent string, layout int) string {
	a := p.v.([]*Value)
	fmt := ""
	max := len(a) - 1
	for i, it := range a {

		if !it.IsValid() {
			return InvalidItem.Error()
		}

		if layout == 1 && len(it.multiComments) != 0 {
			fmt += "\n"
			for _, s := range it.multiComments {
				fmt += indent + s + "\n"
			}
		}

		if i != max {
			fmt += it.string(indent, layout) + ", "
		} else {
			fmt += it.string(indent, layout)
		}

		if layout == 1 && it.eolComment != "" {
			fmt += it.eolComment + "\n" + indent
		}
	}

	return "[" + fmt + "]"
}

func (it Item) Apply(dst interface{}) (count int) {
	if !it.IsValid() {
		return
	}
	return it.Value.Apply(dst)
}

func (it *Value) Apply(dst interface{}) (count int) {
	if !it.IsValid() {
		return
	}

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

	if !vv.IsValid() || !vv.CanSet() {
		return
	}

	return it.apply(vv)
}

func (it *Value) apply(vv reflect.Value) (count int) {

	vt := vv.Type()

	switch vt.Kind() {
	case reflect.Bool:
		if it.kind == Boolean {
			vv.SetBool(it.Boolean())
			count++
		}
	case reflect.String:
		if it.kind >= String && it.kind < StringArray {
			vv.SetString(it.String())
			count++
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if it.kind == Integer {
			vv.SetInt(it.Int())
			count++
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if it.kind == Integer {
			vv.SetUint(it.UInt())
			count++
		}
	case reflect.Float32, reflect.Float64:
		if it.kind == Float {
			vv.SetFloat(it.Float())
			count++
		}
	case reflect.Interface:
		if vt.String() == "interface {}" && it.IsValid() {
			vv.Set(reflect.ValueOf(it.v))
			count++
		}
	case reflect.Struct:
		if it.IsValid() && it.kind == Datetime && vt.String() == "time.Time" {
			vv.Set(reflect.ValueOf(it.Datetime()))
			count++
		}
	case reflect.Array, reflect.Slice:

		l := it.Len()
		if l <= 0 {
			break
		}

		if vt.Kind() == reflect.Slice && vv.Len() < l {

			// ? How to reflect.NewAt(typ, p)

			if vv.Cap() < l {
				l = vv.Cap()
			}
			vv.SetLen(l)
		}

		for i := 0; i < l && i < vv.Len(); i++ {
			count += it.Index(i).apply(vv.Index(i))
		}
	}
	return
}
