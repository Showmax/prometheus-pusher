/**
硬编码实现 PEG.
*/
package toml

import (
	"strconv"
)

const (
	// 发给 itsToken 的指令
	cTokenReset = -1 - iota
	cTokenName  // 返回 token 用于判断名称
)

type itsToken func(rune, int, bool) (Status, Token)

func (i itsToken) Token() Token {
	if i == nil {
		return tokenNothing
	}
	_, token := i(0, cTokenName, true)
	return token
}

func (i itsToken) Reset() {
	if i != nil {
		i(0, cTokenReset, true)
	}
}

type stager interface {
	// 场景名
	Name() string
	// 场景全名
	String() string
	// 返回场景角色
	Roles() []role
	// 通过 当前的 Token 和 当前的 stager 返回下一个 stager
	Next(Token, stager) stager
	// 没有匹配的情况下, 执行 Must, 返回非 nil stager .
	Must(rune) (Status, Token, stager)
}

const (
	stageEnd constStage = iota
	stageInvalid
	stageError
)

var constStagesName = [...]string{
	"stageEnd",
	"stageInvalid",
	"stageError",
}

// 常量场景, 用于标记特殊状态
type constStage int

func (s constStage) String() string {
	if int(s) < len(constStagesName) {
		return constStagesName[s]
	}
	return "stage." + strconv.Itoa(int(s))
}
func (s constStage) Name() string {
	return s.String()
}
func (s constStage) Roles() []role             { return nil }
func (s constStage) Next(Token, stager) stager { return s }
func (s constStage) Must(rune) (Status, Token, stager) {
	return SNot, tokenNothing, stageError
}

// 角色(rules?)
type role struct {
	/**
	Is 判断角色是否能识别传入的字符.
	返回值:
		Status 识别状态
		Token  此值被复用, 如果 Status 是 SMaybe, 会作为 flag 的值.
	参数:
		char  是待判定字符
		flag  是上一次 SMaybe 状态的 uint(Token), 默认为 0.
		race  表示是否有其他 role 竞争
	*/
	Is itsToken
	// 与角色绑定的预期 stager
	// nil 值表示保持 当前的 stager 不变
	Stager stager
}

// 基本场景, 固定的, 不变的
type stage struct {
	name  string
	roles []role
}

func (s stage) String() string {
	if s.name != "" {
		return s.name
	}
	return "UnnamedStage"
}

func (s stage) Name() string {
	return s.String()
}

func (s stage) Roles() []role {
	roles := make([]role, len(s.roles))
	copy(roles, s.roles)
	return roles
	for _, role := range roles {
		role.Is.Reset()
	}
	return roles
}

func (s stage) Next(token Token, stage stager) stager {
	return s
}

func (s stage) Must(rune) (Status, Token, stager) {
	return SNot, tokenNothing, stageInvalid
}

/**
firstStage 可用于第一个场景, NotMatch 方法中对 Roles 进行自循环
*/
type firstStage struct {
	stage
}

func (s firstStage) Must(char rune) (Status, Token, stager) {
	roles := s.Roles()
	for _, role := range roles {
		role.Is.Reset()
		status, token := role.Is(char, 0, false)
		if status == SYes || status == SYesKeep {
			return status, token, role.Stager
		}
	}
	return SNot, tokenNothing, stageInvalid
}

/**
升降场景, 升上去总要降下来, 用于嵌套情况, 比如数组.
成员 stager 提供 roles.
当 level 为 0 回退到 back 场景.
Next 被调用时 level+1.
Must 被调用时决定是否 level-1,
因只有数组这一种情况, 判断函数就是 itsArrayRightBrack.
当然如果 itsArrayRightBrack 不返回 SYes, 那一定是出错了.
*/
type liftStage struct {
	stager // stageArray
	//back    stager
	closed bool
	must   itsToken
}

func (s liftStage) String() string {
	return "liftStage." + s.stager.String()
}

func (s liftStage) Next(token Token, current stager) stager {
	s.stager = current
	return s
}

func (s liftStage) Must(r rune) (Status, Token, stager) {
	status, token := s.must(r, 0, false)
	if status == SYes || status == SYesKeep {
		if token == tokenArrayRightBrack {
			if s.closed {
				return status, token, stageError
			}
			s.closed = true
			return status, token, s
		}
		return status, token, s.stager
	}
	if s.closed {
		s.must.Reset()
		return s.stager.Must(r)
	}
	return SInvalid, tokenNothing, stageInvalid
}

// 回退场景, stager 提供 roles,
// back 开始应该为 nil, 由 Next 进行设置, 下一次 Next 返回 back
// Token close 描述 toles 中要替换的 stager, TOML 中只有 tokenArrayRightBrack
type backStage struct {
	stager
	back stager
	must itsToken
}

func (s backStage) String() string {
	return "backStage." + s.stager.String()
}
func (s backStage) Next(token Token, current stager) stager {
	if s.back == nil {
		s.back = current
	}
	return s
}

func (s backStage) Must(char rune) (Status, Token, stager) {
	if s.must == nil {
		return s.back.Must(char)
	}
	status, token := s.must(char, 0, false)
	if status == SYes || status == SYesKeep {
		return status, token, s.back
	}
	return SInvalid, tokenNothing, stageInvalid
}

// 角色环(token 环)
func rolesCircle(fns ...itsToken) itsToken {
	max := len(fns)
	if max == 0 {
		return func(char rune, flag int, race bool) (Status, Token) { return SNot, tokenNothing }
	}
	i := 0
	return func(char rune, flag int, race bool) (Status, Token) {
		var (
			s Status
			t Token
		)
		if flag == cTokenReset {
			i = 0 // 清零, 新循环开始了
			return SNot, tokenNothing
		}
		if i == max {
			i = 0
		}
		s, t = fns[i](char, flag, race)
		if flag == cTokenName {
			return s, t
		}
		if s == SYes || s == SYesKeep {
			i++
		}

		return s, t
	}
}

/**
跟屁虫 token, f1 要先通过一次之后, f1, f2 顺序尝试
*/
func rolesYesman(f1, f2 itsToken) itsToken {
	yes := false
	return func(char rune, flag int, race bool) (Status, Token) {
		if flag == cTokenReset {
			yes = false // 清零, 新循环开始了
			return SNot, tokenNothing
		}
		s, t := f1(char, flag, race)
		if flag == cTokenName {
			return s, t
		}
		if s == SYes || s == SYesKeep {
			yes = true
			return s, t
		}
		if !yes {
			return s, t
		}

		return f2(char, flag, race)
	}
}

/**
角色粉, f1, f2 顺序尝试, 如果 f1 没有要先通过一次, f2 被匹配, 返回 SInvalid
用例: 多维数组 [[...],[...]] 总是先有 "]", 如果先出现 , 那就 非法了
*/
func rolesFans(f1, f2 itsToken) itsToken {
	yes := false
	return func(char rune, flag int, race bool) (Status, Token) {
		if flag == cTokenReset {
			//yes = false // 清零, 新循环开始了
			return SNot, tokenNothing
		}
		s, t := f1(char, flag, race)
		if flag == cTokenName {
			return s, t
		}
		if s == SYes || s == SYesKeep {
			yes = true
			return s, t
		}

		s, t = f2(char, flag, race)

		if !yes && (s == SYes || s == SYesKeep) {
			return SUnexpected, t
		}
		return s, t
	}
}

// 开启新舞台, 返回第一个场景
func openStage() stager {
	stageEmpty := &firstStage{stage{name: "stageEmpty"}}
	stageEqual := &stage{name: "stageEqual"}
	stageValues := &stage{name: "stageValues"}
	stageArray := &stage{name: "stageArray"}
	stageStringArray := &stage{name: "stageStringArray"}
	stageBooleanArray := &stage{name: "stageBooleanArray"}
	stageIntegerArray := &stage{name: "stageIntegerArray"}
	stageFloatArray := &stage{name: "stageFloatArray"}
	stageDatetimeArray := &stage{name: "stageDatetimeArray"}

	stageEmpty.roles = []role{
		{itsEOF, stageEnd},
		{itsWhitespace, nil},
		{itsNewLine, nil},
		{itsComment, nil},
		{itsTableName, nil},
		{itsArrayOfTables, nil},
		{itsKey, stageEqual},
	}
	// Key = 其实是完全匹配 token 序列.
	stageEqual.roles = []role{
		{itsWhitespace, nil},
		{itsEqual, stageValues},
	}

	stageValues.roles = []role{
		{itsWhitespace, nil},
		{itsArrayLeftBrack,
			backStage{stageArray, stageEmpty, itsArrayRightBrack}},
		{itsString, stageEmpty},
		{itsBoolean, stageEmpty},
		{itsInteger, stageEmpty},
		{itsFloat, stageEmpty},
		{itsDatetime, stageEmpty},
	}

	stageArray.roles = []role{
		{itsWhitespace, nil},
		{itsComment, nil},
		{itsNewLine, nil},
		{itsArrayLeftBrack,
			liftStage{nil, false,
				rolesCircle(itsArrayRightBrack, itsComma)}},
		{itsString,
			backStage{stageStringArray, nil, nil}},
		{itsBoolean,
			backStage{stageBooleanArray, nil, nil}},
		{itsInteger,
			backStage{stageIntegerArray, nil, nil}},
		{itsFloat,
			backStage{stageFloatArray, nil, nil}},
		{itsDatetime,
			backStage{stageDatetimeArray, nil, nil}},
	}

	stageStringArray.roles = []role{
		{itsWhitespace, nil},
		{itsNewLine, nil},
		{itsComment, nil},
		{rolesCircle(itsComma, itsString), nil},
	}

	stageBooleanArray.roles = []role{
		{itsWhitespace, nil},
		{itsNewLine, nil},
		{itsComment, nil},
		{rolesCircle(itsComma, itsBoolean), nil},
	}

	stageIntegerArray.roles = []role{
		{itsWhitespace, nil},
		{itsNewLine, nil},
		{itsComment, nil},
		{rolesCircle(itsComma, itsInteger), nil},
	}

	stageFloatArray.roles = []role{
		{itsWhitespace, nil},
		{itsNewLine, nil},
		{itsComment, nil},
		{rolesCircle(itsComma, itsFloat), nil},
	}

	stageDatetimeArray.roles = []role{
		{itsWhitespace, nil},
		{itsNewLine, nil},
		{itsComment, nil},
		{rolesCircle(itsComma, itsDatetime), nil},
	}

	return stageEmpty
}

func stagePlay(p parser, stage stager) {
Loop:
	for stage != nil {
		if stage == stageEnd {
			break
		}
		if stage == stageInvalid {
			p.Invalid(tokenError)
			break
		}
		if stage == stageError {
			p.Err(stage.String())
			break
		}

		roles := stage.Roles()
		skip := make([]bool, len(roles))
		flag := make([]int, len(roles))

		if len(roles) == 0 {
			p.Invalid(tokenError)
			break
		}

		var (
			st    Status
			token Token // flag 是 uint 和 Token 复用
			maybe int
			r     rune
		)

		for {

			r = p.Next()
			if r == RuneError {
				p.Invalid(tokenRuneError)
				return
			}
			for i, role := range roles {
				if skip[i] {
					continue
				}
				st, token = role.Is(r, flag[i], maybe != 0)
				switch st {
				case SMaybe:
					if flag[i] == 0 {
						maybe++
					}
					flag[i] = int(token)

				case SYes, SYesKeep:
					if st == SYesKeep {
						p.Keep()
					}
					if p.Token(token) != nil {
						return
					}

					if role.Stager != nil {
						stage = role.Stager.Next(token, stage)
					}
					continue Loop

				case SNot:

					if flag[i] != 0 {
						maybe--
					}
					skip[i] = true

				case SInvalid:
					p.Invalid(token)
					return
				}
			}
			if maybe != 0 && r != EOF {
				continue
			}

			stageName := stage.Name()
			st, token, stage = stage.Must(r)

			if stage == nil || st != SYes && st != SYesKeep {
				if st == SUnexpected {
					p.Err("unexpercted " + token.String() + " of " + stageName)
				} else {
					p.Err("roles does not match one of " + stageName)
				}
				return
			}
			if st == SYesKeep {
				p.Keep()
			}
			if p.Token(token) != nil {
				return
			}
			break
		}
	}

}
