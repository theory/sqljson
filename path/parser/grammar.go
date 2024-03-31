// Code generated by goyacc -v  -o grammar.go -p path grammar.y. DO NOT EDIT.

//line grammar.y:2
/*-------------------------------------------------------------------------
 *
 * grammar.y
 *	 Grammar definitions for jsonpath datatype
 *
 * Transforms tokenized jsonpath into tree of JsonPathParseItem structs.
 *
 * Copyright (c) 2019-2024, PostgreSQL Global Development Group
 *
 * IDENTIFICATION
 *	https://git.postgresql.org/gitweb/?p=postgresql.git;a=blob;f=src/backend/utils/adt/jsonpath_gram.y;h=8733a0e;hb=HEAD
 *
 *-------------------------------------------------------------------------
 */

package parser

import __yyfmt__ "fmt"

//line grammar.y:17

import (
	"strconv"

	"github.com/theory/sqljson/path/ast"
)

//line grammar.y:26
type pathSymType struct {
	yys     int
	str     string
	elems   []ast.Node
	indexs  []ast.Node
	value   ast.Node
	optype  ast.BinaryOperator
	method  ast.MethodNode
	boolean bool
	integer int
}

const TO_P = 57346
const NULL_P = 57347
const TRUE_P = 57348
const FALSE_P = 57349
const IS_P = 57350
const UNKNOWN_P = 57351
const EXISTS_P = 57352
const IDENT_P = 57353
const STRING_P = 57354
const NUMERIC_P = 57355
const INT_P = 57356
const VARIABLE_P = 57357
const OR_P = 57358
const AND_P = 57359
const NOT_P = 57360
const LESS_P = 57361
const LESSEQUAL_P = 57362
const EQUAL_P = 57363
const NOTEQUAL_P = 57364
const GREATEREQUAL_P = 57365
const GREATER_P = 57366
const ANY_P = 57367
const STRICT_P = 57368
const LAX_P = 57369
const LAST_P = 57370
const STARTS_P = 57371
const WITH_P = 57372
const LIKE_REGEX_P = 57373
const FLAG_P = 57374
const ABS_P = 57375
const SIZE_P = 57376
const TYPE_P = 57377
const FLOOR_P = 57378
const DOUBLE_P = 57379
const CEILING_P = 57380
const KEYVALUE_P = 57381
const DATETIME_P = 57382
const BIGINT_P = 57383
const BOOLEAN_P = 57384
const DATE_P = 57385
const DECIMAL_P = 57386
const INTEGER_P = 57387
const NUMBER_P = 57388
const STRINGFUNC_P = 57389
const TIME_P = 57390
const TIME_TZ_P = 57391
const TIMESTAMP_P = 57392
const TIMESTAMP_TZ_P = 57393
const UMINUS = 57394

var pathToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"TO_P",
	"NULL_P",
	"TRUE_P",
	"FALSE_P",
	"IS_P",
	"UNKNOWN_P",
	"EXISTS_P",
	"IDENT_P",
	"STRING_P",
	"NUMERIC_P",
	"INT_P",
	"VARIABLE_P",
	"OR_P",
	"AND_P",
	"NOT_P",
	"LESS_P",
	"LESSEQUAL_P",
	"EQUAL_P",
	"NOTEQUAL_P",
	"GREATEREQUAL_P",
	"GREATER_P",
	"ANY_P",
	"STRICT_P",
	"LAX_P",
	"LAST_P",
	"STARTS_P",
	"WITH_P",
	"LIKE_REGEX_P",
	"FLAG_P",
	"ABS_P",
	"SIZE_P",
	"TYPE_P",
	"FLOOR_P",
	"DOUBLE_P",
	"CEILING_P",
	"KEYVALUE_P",
	"DATETIME_P",
	"BIGINT_P",
	"BOOLEAN_P",
	"DATE_P",
	"DECIMAL_P",
	"INTEGER_P",
	"NUMBER_P",
	"STRINGFUNC_P",
	"TIME_P",
	"TIME_TZ_P",
	"TIMESTAMP_P",
	"TIMESTAMP_TZ_P",
	"'+'",
	"'-'",
	"'*'",
	"'/'",
	"'%'",
	"UMINUS",
	"'('",
	"')'",
	"'$'",
	"'@'",
	"','",
	"'['",
	"']'",
	"'{'",
	"'}'",
	"'.'",
	"'?'",
}

var pathStatenames = [...]string{}

const pathEofCode = 1
const pathErrCode = 2
const pathInitialStackSize = 16

//line grammar.y:331

var pathExca = [...]int16{
	-1, 1,
	1, -1,
	-2, 0,
	-1, 79,
	58, 122,
	-2, 98,
	-1, 80,
	58, 123,
	-2, 99,
	-1, 81,
	58, 124,
	-2, 100,
	-1, 82,
	58, 125,
	-2, 101,
	-1, 83,
	58, 126,
	-2, 102,
	-1, 84,
	58, 127,
	-2, 103,
	-1, 85,
	58, 128,
	-2, 105,
	-1, 86,
	58, 129,
	-2, 111,
	-1, 87,
	58, 130,
	-2, 112,
	-1, 88,
	58, 131,
	-2, 113,
	-1, 89,
	58, 132,
	-2, 115,
	-1, 90,
	58, 133,
	-2, 116,
	-1, 91,
	58, 134,
	-2, 117,
}

const pathPrivate = 57344

const pathLast = 250

var pathAct = [...]uint8{
	158, 145, 65, 111, 152, 6, 136, 178, 132, 7,
	133, 131, 49, 50, 52, 43, 47, 129, 166, 48,
	44, 46, 175, 173, 30, 31, 32, 33, 34, 172,
	56, 140, 171, 59, 60, 61, 62, 63, 42, 41,
	170, 169, 165, 37, 39, 35, 36, 40, 38, 142,
	112, 64, 66, 28, 49, 29, 128, 127, 117, 126,
	125, 115, 124, 123, 116, 94, 95, 96, 97, 98,
	99, 100, 92, 93, 42, 41, 30, 31, 32, 33,
	34, 161, 15, 114, 174, 122, 78, 101, 102, 103,
	104, 105, 106, 107, 79, 80, 81, 82, 83, 84,
	85, 72, 86, 87, 88, 71, 89, 90, 91, 73,
	74, 75, 76, 108, 55, 68, 121, 139, 130, 57,
	146, 137, 41, 42, 41, 168, 42, 41, 135, 151,
	54, 155, 156, 157, 141, 112, 162, 163, 21, 22,
	23, 3, 4, 15, 167, 20, 24, 25, 26, 134,
	154, 13, 32, 33, 34, 21, 22, 23, 147, 148,
	58, 19, 20, 24, 25, 26, 138, 164, 176, 113,
	159, 77, 21, 22, 23, 2, 177, 70, 19, 20,
	24, 25, 26, 47, 160, 10, 11, 44, 46, 42,
	41, 9, 27, 17, 18, 19, 110, 30, 31, 32,
	33, 34, 10, 11, 109, 143, 119, 12, 51, 120,
	17, 18, 37, 39, 35, 36, 40, 38, 144, 10,
	11, 53, 28, 8, 29, 51, 153, 17, 18, 30,
	31, 32, 33, 34, 149, 150, 5, 118, 67, 69,
	45, 14, 16, 1, 0, 30, 31, 32, 33, 34,
}

var pathPact = [...]int16{
	115, -1000, 133, -1000, -1000, -1000, 193, 173, -47, 133,
	167, 167, -1000, 72, -1000, 56, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, 167, 89, 148,
	167, 167, 167, 167, 167, -1000, -1000, -1000, -1000, -1000,
	-1000, 133, 133, -1000, 61, -1000, 55, 150, 110, 24,
	-1000, 133, -1000, -1000, 133, 167, 177, 194, 84, 98,
	98, -1000, -1000, -1000, -1000, 193, 105, -1000, -1000, -1000,
	27, 5, 4, 2, 1, -1, -2, -1000, -48, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, 133, -53,
	-54, -1000, 145, 120, -47, 107, 58, -28, -1000, -1000,
	-1000, 122, -10, 106, 117, 136, 136, 136, 136, 156,
	22, -1000, 167, -1000, 167, 158, -1000, -1000, -47, -1000,
	-1000, -1000, -1000, -17, -44, -1000, -1000, 130, 111, -18,
	-1000, -1000, -19, -1000, -1000, -27, -30, -36, 18, -1000,
	-1000, -1000, -1000, 177, -1000, -1000, 106, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, 156, -1000, -59, -1000,
}

var pathPgo = [...]uint8{
	0, 243, 242, 241, 2, 240, 239, 6, 238, 9,
	207, 3, 237, 236, 235, 234, 1, 226, 4, 223,
	218, 205, 196, 192, 177, 175, 171, 0,
}

var pathR1 = [...]int8{
	0, 1, 13, 13, 25, 25, 25, 2, 2, 2,
	2, 2, 2, 2, 23, 23, 23, 23, 23, 23,
	10, 10, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 12, 12, 3, 3, 3, 3, 19, 19, 19,
	19, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	11, 11, 22, 22, 5, 5, 27, 27, 6, 6,
	6, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 16, 16, 16, 20, 20, 21, 21,
	17, 18, 18, 14, 15, 15, 8, 26, 26, 26,
	26, 26, 26, 26, 26, 26, 26, 26, 26, 26,
	26, 26, 26, 26, 26, 26, 26, 26, 26, 26,
	26, 26, 26, 26, 26, 26, 26, 26, 26, 26,
	26, 26, 24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24,
}

var pathR2 = [...]int8{
	0, 2, 1, 1, 1, 1, 0, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	3, 4, 1, 3, 3, 3, 2, 5, 4, 3,
	5, 1, 1, 1, 1, 1, 1, 1, 4, 4,
	2, 1, 3, 2, 2, 3, 3, 3, 3, 3,
	1, 3, 1, 3, 3, 3, 1, 1, 1, 4,
	6, 2, 2, 1, 2, 4, 4, 5, 5, 5,
	5, 5, 5, 1, 2, 2, 1, 3, 1, 0,
	1, 1, 0, 1, 1, 0, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1,
}

var pathChk = [...]int16{
	-1000, -1, -25, 26, 27, -13, -4, -9, -19, 58,
	52, 53, -10, 18, -3, 10, -2, 60, 61, 28,
	12, 5, 6, 7, 13, 14, 15, -23, 29, 31,
	52, 53, 54, 55, 56, 21, 22, 19, 24, 20,
	23, 17, 16, -7, 67, -5, 68, 63, -9, -4,
	-4, 58, -4, -10, 58, 58, -4, 30, 12, -4,
	-4, -4, -4, -4, -9, -4, -9, -8, 54, -6,
	-24, 44, 40, 48, 49, 50, 51, -26, 25, 33,
	34, 35, 36, 37, 38, 39, 41, 42, 43, 45,
	46, 47, 11, 12, 4, 5, 6, 7, 8, 9,
	10, 26, 27, 28, 29, 30, 31, 32, 58, 54,
	-22, -11, -4, 59, 59, -9, -9, -4, -12, 12,
	15, 32, 58, 58, 58, 58, 58, 58, 58, 65,
	-9, 64, 62, 64, 4, 8, -7, -7, 59, 59,
	59, 12, 59, -21, -20, -16, 14, 52, 53, -15,
	-14, 12, -18, -17, 14, -18, -18, -18, -27, 14,
	28, 59, -11, -4, 9, 59, 62, 14, 14, 59,
	59, 59, 59, 59, 66, 4, -16, -27, 66,
}

var pathDef = [...]int8{
	6, -2, 0, 4, 5, 1, 2, 3, 41, 0,
	0, 0, 22, 0, 37, 0, 33, 34, 35, 36,
	7, 8, 9, 10, 11, 12, 13, 0, 0, 0,
	0, 0, 0, 0, 0, 14, 15, 16, 17, 18,
	19, 0, 0, 40, 0, 63, 0, 0, 0, 0,
	43, 0, 44, 26, 0, 0, 23, 0, 29, 45,
	46, 47, 48, 49, 24, 0, 25, 61, 62, 64,
	0, 114, 104, 118, 119, 120, 121, 86, 58, -2,
	-2, -2, -2, -2, -2, -2, -2, -2, -2, -2,
	-2, -2, 87, 88, 89, 90, 91, 92, 93, 94,
	95, 96, 97, 106, 107, 108, 109, 110, 0, 0,
	0, 52, 50, 20, 42, 0, 0, 0, 28, 31,
	32, 0, 0, 79, 85, 82, 82, 82, 82, 0,
	0, 54, 0, 55, 0, 0, 39, 38, 0, 20,
	21, 30, 65, 0, 78, 76, 73, 0, 0, 0,
	84, 83, 0, 81, 80, 0, 0, 0, 0, 56,
	57, 66, 53, 51, 27, 67, 0, 74, 75, 68,
	69, 70, 71, 72, 59, 0, 77, 0, 60,
}

var pathTok1 = [...]int8{
	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 60, 56, 3, 3,
	58, 59, 54, 52, 62, 53, 67, 55, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 68, 61, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 63, 3, 64, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 65, 3, 66,
}

var pathTok2 = [...]int8{
	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40, 41,
	42, 43, 44, 45, 46, 47, 48, 49, 50, 51,
	57,
}

var pathTok3 = [...]int8{
	0,
}

var pathErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}


/*	parser for yacc output	*/

var (
	pathDebug        = 0
	pathErrorVerbose = false
)

type pathLexer interface {
	Lex(lval *pathSymType) int
	Error(s string)
}

type pathParser interface {
	Parse(pathLexer) int
	Lookahead() int
}

type pathParserImpl struct {
	lval  pathSymType
	stack [pathInitialStackSize]pathSymType
	char  int
}

func (p *pathParserImpl) Lookahead() int {
	return p.char
}

func pathNewParser() pathParser {
	return &pathParserImpl{}
}

const pathFlag = -1000

func pathTokname(c int) string {
	if c >= 1 && c-1 < len(pathToknames) {
		if pathToknames[c-1] != "" {
			return pathToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func pathStatname(s int) string {
	if s >= 0 && s < len(pathStatenames) {
		if pathStatenames[s] != "" {
			return pathStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func pathErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !pathErrorVerbose {
		return "syntax error"
	}

	for _, e := range pathErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + pathTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := int(pathPact[state])
	for tok := TOKSTART; tok-1 < len(pathToknames); tok++ {
		if n := base + tok; n >= 0 && n < pathLast && int(pathChk[int(pathAct[n])]) == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if pathDef[state] == -2 {
		i := 0
		for pathExca[i] != -1 || int(pathExca[i+1]) != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; pathExca[i] >= 0; i += 2 {
			tok := int(pathExca[i])
			if tok < TOKSTART || pathExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if pathExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += pathTokname(tok)
	}
	return res
}

func pathlex1(lex pathLexer, lval *pathSymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = int(pathTok1[0])
		goto out
	}
	if char < len(pathTok1) {
		token = int(pathTok1[char])
		goto out
	}
	if char >= pathPrivate {
		if char < pathPrivate+len(pathTok2) {
			token = int(pathTok2[char-pathPrivate])
			goto out
		}
	}
	for i := 0; i < len(pathTok3); i += 2 {
		token = int(pathTok3[i+0])
		if token == char {
			token = int(pathTok3[i+1])
			goto out
		}
	}

out:
	if token == 0 {
		token = int(pathTok2[1]) /* unknown char */
	}
	if pathDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", pathTokname(token), uint(char))
	}
	return char, token
}

func pathParse(pathlex pathLexer) int {
	return pathNewParser().Parse(pathlex)
}

func (pathrcvr *pathParserImpl) Parse(pathlex pathLexer) int {
	var pathn int
	var pathVAL pathSymType
	var pathDollar []pathSymType
	_ = pathDollar // silence set and not used
	pathS := pathrcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	pathstate := 0
	pathrcvr.char = -1
	pathtoken := -1 // pathrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		pathstate = -1
		pathrcvr.char = -1
		pathtoken = -1
	}()
	pathp := -1
	goto pathstack

ret0:
	return 0

ret1:
	return 1

pathstack:
	/* put a state and value onto the stack */
	if pathDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", pathTokname(pathtoken), pathStatname(pathstate))
	}

	pathp++
	if pathp >= len(pathS) {
		nyys := make([]pathSymType, len(pathS)*2)
		copy(nyys, pathS)
		pathS = nyys
	}
	pathS[pathp] = pathVAL
	pathS[pathp].yys = pathstate

pathnewstate:
	pathn = int(pathPact[pathstate])
	if pathn <= pathFlag {
		goto pathdefault /* simple state */
	}
	if pathrcvr.char < 0 {
		pathrcvr.char, pathtoken = pathlex1(pathlex, &pathrcvr.lval)
	}
	pathn += pathtoken
	if pathn < 0 || pathn >= pathLast {
		goto pathdefault
	}
	pathn = int(pathAct[pathn])
	if int(pathChk[pathn]) == pathtoken { /* valid shift */
		pathrcvr.char = -1
		pathtoken = -1
		pathVAL = pathrcvr.lval
		pathstate = pathn
		if Errflag > 0 {
			Errflag--
		}
		goto pathstack
	}

pathdefault:
	/* default state action */
	pathn = int(pathDef[pathstate])
	if pathn == -2 {
		if pathrcvr.char < 0 {
			pathrcvr.char, pathtoken = pathlex1(pathlex, &pathrcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if pathExca[xi+0] == -1 && int(pathExca[xi+1]) == pathstate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			pathn = int(pathExca[xi+0])
			if pathn < 0 || pathn == pathtoken {
				break
			}
		}
		pathn = int(pathExca[xi+1])
		if pathn < 0 {
			goto ret0
		}
	}
	if pathn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			pathlex.Error(pathErrorMessage(pathstate, pathtoken))
			Nerrs++
			if pathDebug >= 1 {
				__yyfmt__.Printf("%s", pathStatname(pathstate))
				__yyfmt__.Printf(" saw %s\n", pathTokname(pathtoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for pathp >= 0 {
				pathn = int(pathPact[pathS[pathp].yys]) + pathErrCode
				if pathn >= 0 && pathn < pathLast {
					pathstate = int(pathAct[pathn]) /* simulate a shift of "error" */
					if int(pathChk[pathstate]) == pathErrCode {
						goto pathstack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if pathDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", pathS[pathp].yys)
				}
				pathp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if pathDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", pathTokname(pathtoken))
			}
			if pathtoken == pathEofCode {
				goto ret1
			}
			pathrcvr.char = -1
			pathtoken = -1
			goto pathnewstate /* try again in the same state */
		}
	}

	/* reduction by production pathn */
	if pathDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", pathn, pathStatname(pathstate))
	}

	pathnt := pathn
	pathpt := pathp
	_ = pathpt // guard against "declared and not used"

	pathp -= int(pathR2[pathn])
	// pathp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if pathp+1 >= len(pathS) {
		nyys := make([]pathSymType, len(pathS)*2)
		copy(nyys, pathS)
		pathS = nyys
	}
	pathVAL = pathS[pathp+1]

	/* consult goto table to find next state */
	pathn = int(pathR1[pathn])
	pathg := int(pathPgo[pathn])
	pathj := pathg + pathS[pathp].yys + 1

	if pathj >= pathLast {
		pathstate = int(pathAct[pathg])
	} else {
		pathstate = int(pathAct[pathj])
		if int(pathChk[pathstate]) != -pathn {
			pathstate = int(pathAct[pathg])
		}
	}
	// dummy call; replaced with literal code
	switch pathnt {

	case 1:
		pathDollar = pathS[pathpt-2 : pathpt+1]
//line grammar.y:81
		{
			pathlex.(*lexer).setResult(pathDollar[1].boolean, pathDollar[2].value)
		}
	case 2:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:85
		{
			pathVAL.value = pathDollar[1].value
		}
	case 3:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:86
		{
			pathVAL.value = pathDollar[1].value
		}
	case 4:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:90
		{
			pathVAL.boolean = false
		}
	case 5:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:91
		{
			pathVAL.boolean = true
		}
	case 6:
		pathDollar = pathS[pathpt-0 : pathpt+1]
//line grammar.y:92
		{
			pathVAL.boolean = true
		}
	case 7:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:96
		{
			pathVAL.value = ast.NewString(pathDollar[1].str)
		}
	case 8:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:97
		{
			pathVAL.value = ast.ConstNull
		}
	case 9:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:98
		{
			pathVAL.value = ast.ConstTrue
		}
	case 10:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:99
		{
			pathVAL.value = ast.ConstFalse
		}
	case 11:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:100
		{
			pathVAL.value = ast.NewNumeric(pathDollar[1].str)
		}
	case 12:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:101
		{
			pathVAL.value = ast.NewInteger(pathDollar[1].str)
		}
	case 13:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:102
		{
			pathVAL.value = ast.NewVariable(pathDollar[1].str)
		}
	case 14:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:106
		{
			pathVAL.optype = ast.BinaryEqual
		}
	case 15:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:107
		{
			pathVAL.optype = ast.BinaryNotEqual
		}
	case 16:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:108
		{
			pathVAL.optype = ast.BinaryLess
		}
	case 17:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:109
		{
			pathVAL.optype = ast.BinaryGreater
		}
	case 18:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:110
		{
			pathVAL.optype = ast.BinaryLessOrEqual
		}
	case 19:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:111
		{
			pathVAL.optype = ast.BinaryGreaterOrEqual
		}
	case 20:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:115
		{
			pathVAL.value = pathDollar[2].value
		}
	case 21:
		pathDollar = pathS[pathpt-4 : pathpt+1]
//line grammar.y:116
		{
			pathVAL.value = ast.NewUnary(ast.UnaryExists, pathDollar[3].value)
		}
	case 22:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:120
		{
			pathVAL.value = pathDollar[1].value
		}
	case 23:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:121
		{
			pathVAL.value = ast.NewBinary(pathDollar[2].optype, pathDollar[1].value, pathDollar[3].value)
		}
	case 24:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:122
		{
			pathVAL.value = ast.NewBinary(ast.BinaryAnd, pathDollar[1].value, pathDollar[3].value)
		}
	case 25:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:123
		{
			pathVAL.value = ast.NewBinary(ast.BinaryOr, pathDollar[1].value, pathDollar[3].value)
		}
	case 26:
		pathDollar = pathS[pathpt-2 : pathpt+1]
//line grammar.y:124
		{
			pathVAL.value = ast.NewUnary(ast.UnaryNot, pathDollar[2].value)
		}
	case 27:
		pathDollar = pathS[pathpt-5 : pathpt+1]
//line grammar.y:126
		{
			pathVAL.value = ast.NewUnary(ast.UnaryIsUnknown, pathDollar[2].value)
		}
	case 28:
		pathDollar = pathS[pathpt-4 : pathpt+1]
//line grammar.y:128
		{
			pathVAL.value = ast.NewBinary(ast.BinaryStartsWith, pathDollar[1].value, pathDollar[4].value)
		}
	case 29:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:130
		{
			var err error
			pathVAL.value, err = ast.NewRegex(pathDollar[1].value, pathDollar[3].str, "")
			if err != nil {
				pathlex.Error(err.Error())
			}
		}
	case 30:
		pathDollar = pathS[pathpt-5 : pathpt+1]
//line grammar.y:138
		{
			var err error
			pathVAL.value, err = ast.NewRegex(pathDollar[1].value, pathDollar[3].str, pathDollar[5].str)
			if err != nil {
				pathlex.Error(err.Error())
			}
		}
	case 31:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:148
		{
			pathVAL.value = ast.NewString(pathDollar[1].str)
		}
	case 32:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:149
		{
			pathVAL.value = ast.NewVariable(pathDollar[1].str)
		}
	case 33:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:153
		{
			pathVAL.value = pathDollar[1].value
		}
	case 34:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:154
		{
			pathVAL.value = ast.ConstRoot
		}
	case 35:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:155
		{
			pathVAL.value = ast.ConstCurrent
		}
	case 36:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:156
		{
			pathVAL.value = ast.ConstLast
		}
	case 37:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:160
		{
			pathVAL.elems = []ast.Node{pathDollar[1].value}
		}
	case 38:
		pathDollar = pathS[pathpt-4 : pathpt+1]
//line grammar.y:161
		{
			pathVAL.elems = []ast.Node{pathDollar[2].value, pathDollar[4].value}
		}
	case 39:
		pathDollar = pathS[pathpt-4 : pathpt+1]
//line grammar.y:162
		{
			pathVAL.elems = []ast.Node{pathDollar[2].value, pathDollar[4].value}
		}
	case 40:
		pathDollar = pathS[pathpt-2 : pathpt+1]
//line grammar.y:163
		{
			pathVAL.elems = append(pathVAL.elems, pathDollar[2].value)
		}
	case 41:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:167
		{
			pathVAL.value = ast.NewAccessorList(pathDollar[1].elems)
		}
	case 42:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:168
		{
			pathVAL.value = pathDollar[2].value
		}
	case 43:
		pathDollar = pathS[pathpt-2 : pathpt+1]
//line grammar.y:169
		{
			pathVAL.value = ast.NewUnaryOrNumber(ast.UnaryPlus, pathDollar[2].value)
		}
	case 44:
		pathDollar = pathS[pathpt-2 : pathpt+1]
//line grammar.y:170
		{
			pathVAL.value = ast.NewUnaryOrNumber(ast.UnaryMinus, pathDollar[2].value)
		}
	case 45:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:171
		{
			pathVAL.value = ast.NewBinary(ast.BinaryAdd, pathDollar[1].value, pathDollar[3].value)
		}
	case 46:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:172
		{
			pathVAL.value = ast.NewBinary(ast.BinarySub, pathDollar[1].value, pathDollar[3].value)
		}
	case 47:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:173
		{
			pathVAL.value = ast.NewBinary(ast.BinaryMul, pathDollar[1].value, pathDollar[3].value)
		}
	case 48:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:174
		{
			pathVAL.value = ast.NewBinary(ast.BinaryDiv, pathDollar[1].value, pathDollar[3].value)
		}
	case 49:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:175
		{
			pathVAL.value = ast.NewBinary(ast.BinaryMod, pathDollar[1].value, pathDollar[3].value)
		}
	case 50:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:179
		{
			pathVAL.value = ast.NewBinary(ast.BinarySubscript, pathDollar[1].value, nil)
		}
	case 51:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:180
		{
			pathVAL.value = ast.NewBinary(ast.BinarySubscript, pathDollar[1].value, pathDollar[3].value)
		}
	case 52:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:184
		{
			pathVAL.indexs = []ast.Node{pathDollar[1].value}
		}
	case 53:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:185
		{
			pathVAL.indexs = append(pathVAL.indexs, pathDollar[3].value)
		}
	case 54:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:189
		{
			pathVAL.value = ast.ConstAnyArray
		}
	case 55:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:190
		{
			pathVAL.value = ast.NewArrayIndex(pathDollar[2].indexs)
		}
	case 56:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:194
		{
			pathVAL.integer, _ = strconv.Atoi(pathDollar[1].str)
		}
	case 57:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:195
		{
			pathVAL.integer = -1
		}
	case 58:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:199
		{
			pathVAL.value = ast.NewAny(0, -1)
		}
	case 59:
		pathDollar = pathS[pathpt-4 : pathpt+1]
//line grammar.y:200
		{
			pathVAL.value = ast.NewAny(pathDollar[3].integer, pathDollar[3].integer)
		}
	case 60:
		pathDollar = pathS[pathpt-6 : pathpt+1]
//line grammar.y:202
		{
			pathVAL.value = ast.NewAny(pathDollar[3].integer, pathDollar[5].integer)
		}
	case 61:
		pathDollar = pathS[pathpt-2 : pathpt+1]
//line grammar.y:206
		{
			pathVAL.value = pathDollar[2].value
		}
	case 62:
		pathDollar = pathS[pathpt-2 : pathpt+1]
//line grammar.y:207
		{
			pathVAL.value = ast.ConstAnyKey
		}
	case 63:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:208
		{
			pathVAL.value = pathDollar[1].value
		}
	case 64:
		pathDollar = pathS[pathpt-2 : pathpt+1]
//line grammar.y:209
		{
			pathVAL.value = pathDollar[2].value
		}
	case 65:
		pathDollar = pathS[pathpt-4 : pathpt+1]
//line grammar.y:210
		{
			pathVAL.value = pathDollar[2].method
		}
	case 66:
		pathDollar = pathS[pathpt-4 : pathpt+1]
//line grammar.y:211
		{
			pathVAL.value = ast.NewUnary(ast.UnaryFilter, pathDollar[3].value)
		}
	case 67:
		pathDollar = pathS[pathpt-5 : pathpt+1]
//line grammar.y:213
		{
			switch len(pathDollar[4].elems) {
			case 0:
				pathVAL.value = ast.NewBinary(ast.BinaryDecimal, nil, nil)
			case 1:
				pathVAL.value = ast.NewBinary(ast.BinaryDecimal, pathDollar[4].elems[0], nil)
			case 2:
				pathVAL.value = ast.NewBinary(ast.BinaryDecimal, pathDollar[4].elems[0], pathDollar[4].elems[1])
			default:
				panic("invalid input syntax: .decimal() can only have an optional precision[,scale]")
			}
		}
	case 68:
		pathDollar = pathS[pathpt-5 : pathpt+1]
//line grammar.y:226
		{
			pathVAL.value = ast.NewUnary(ast.UnaryDateTime, pathDollar[4].value)
		}
	case 69:
		pathDollar = pathS[pathpt-5 : pathpt+1]
//line grammar.y:228
		{
			pathVAL.value = ast.NewUnary(ast.UnaryTime, pathDollar[4].value)
		}
	case 70:
		pathDollar = pathS[pathpt-5 : pathpt+1]
//line grammar.y:230
		{
			pathVAL.value = ast.NewUnary(ast.UnaryTimeTZ, pathDollar[4].value)
		}
	case 71:
		pathDollar = pathS[pathpt-5 : pathpt+1]
//line grammar.y:232
		{
			pathVAL.value = ast.NewUnary(ast.UnaryTimestamp, pathDollar[4].value)
		}
	case 72:
		pathDollar = pathS[pathpt-5 : pathpt+1]
//line grammar.y:234
		{
			pathVAL.value = ast.NewUnary(ast.UnaryTimestampTZ, pathDollar[4].value)
		}
	case 73:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:239
		{
			pathVAL.value = ast.NewInteger(pathDollar[1].str)
		}
	case 74:
		pathDollar = pathS[pathpt-2 : pathpt+1]
//line grammar.y:241
		{
			pathVAL.value = ast.NewUnaryOrNumber(ast.UnaryPlus, ast.NewInteger(pathDollar[2].str))
		}
	case 75:
		pathDollar = pathS[pathpt-2 : pathpt+1]
//line grammar.y:243
		{
			pathVAL.value = ast.NewUnaryOrNumber(ast.UnaryMinus, ast.NewInteger(pathDollar[2].str))
		}
	case 76:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:247
		{
			pathVAL.elems = []ast.Node{pathDollar[1].value}
		}
	case 77:
		pathDollar = pathS[pathpt-3 : pathpt+1]
//line grammar.y:248
		{
			pathVAL.elems = append(pathVAL.elems, pathDollar[3].value)
		}
	case 78:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:252
		{
			pathVAL.elems = pathDollar[1].elems
		}
	case 79:
		pathDollar = pathS[pathpt-0 : pathpt+1]
//line grammar.y:253
		{
			pathVAL.elems = nil
		}
	case 80:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:257
		{
			pathVAL.value = ast.NewInteger(pathDollar[1].str)
		}
	case 81:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:261
		{
			pathVAL.value = pathDollar[1].value
		}
	case 82:
		pathDollar = pathS[pathpt-0 : pathpt+1]
//line grammar.y:262
		{
			pathVAL.value = nil
		}
	case 83:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:266
		{
			pathVAL.value = ast.NewString(pathDollar[1].str)
		}
	case 84:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:270
		{
			pathVAL.value = pathDollar[1].value
		}
	case 85:
		pathDollar = pathS[pathpt-0 : pathpt+1]
//line grammar.y:271
		{
			pathVAL.value = nil
		}
	case 86:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:275
		{
			pathVAL.value = ast.NewKey(pathDollar[1].str)
		}
	case 122:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:317
		{
			pathVAL.method = ast.MethodAbs
		}
	case 123:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:318
		{
			pathVAL.method = ast.MethodSize
		}
	case 124:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:319
		{
			pathVAL.method = ast.MethodType
		}
	case 125:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:320
		{
			pathVAL.method = ast.MethodFloor
		}
	case 126:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:321
		{
			pathVAL.method = ast.MethodDouble
		}
	case 127:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:322
		{
			pathVAL.method = ast.MethodCeiling
		}
	case 128:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:323
		{
			pathVAL.method = ast.MethodKeyValue
		}
	case 129:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:324
		{
			pathVAL.method = ast.MethodBigint
		}
	case 130:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:325
		{
			pathVAL.method = ast.MethodBoolean
		}
	case 131:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:326
		{
			pathVAL.method = ast.MethodDate
		}
	case 132:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:327
		{
			pathVAL.method = ast.MethodInteger
		}
	case 133:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:328
		{
			pathVAL.method = ast.MethodNumber
		}
	case 134:
		pathDollar = pathS[pathpt-1 : pathpt+1]
//line grammar.y:329
		{
			pathVAL.method = ast.MethodString
		}
	}
	goto pathstack /* stack new state and value */
}
