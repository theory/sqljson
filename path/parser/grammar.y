%{
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

import (
	"strconv"

	"github.com/theory/sqljson/path/ast"
)
%}

%union{
	str     string
	elems   []ast.Node
	indexs  []ast.Node
	value   ast.Node
	optype  ast.BinaryOperator
	method  *ast.MethodNode
	boolean bool
	integer int
}

%token	<str>		TO_P NULL_P TRUE_P FALSE_P IS_P UNKNOWN_P EXISTS_P
%token	<str>		IDENT_P STRING_P NUMERIC_P INT_P VARIABLE_P
%token	<str>		OR_P AND_P NOT_P
%token	<str>		LESS_P LESSEQUAL_P EQUAL_P NOTEQUAL_P GREATEREQUAL_P GREATER_P
%token	<str>		ANY_P STRICT_P LAX_P LAST_P STARTS_P WITH_P LIKE_REGEX_P FLAG_P
%token	<str>		ABS_P SIZE_P TYPE_P FLOOR_P DOUBLE_P CEILING_P KEYVALUE_P
%token	<str>		DATETIME_P
%token	<str>		BIGINT_P BOOLEAN_P DATE_P DECIMAL_P INTEGER_P NUMBER_P
%token	<str>		STRINGFUNC_P TIME_P TIME_TZ_P TIMESTAMP_P TIMESTAMP_TZ_P

%type	<result>	result

%type	<value>		scalar_value path_primary expr array_accessor
					any_path accessor_op key predicate delimited_predicate
					index_elem starts_with_initial expr_or_predicate
					datetime_template opt_datetime_template csv_elem
					datetime_precision opt_datetime_precision

%type	<elems>		accessor_expr csv_list opt_csv_list

%type	<indexs>	index_list

%type	<optype>	comp_op

%type	<method>	method

%type	<boolean>	mode

%type	<str>		key_name

%type	<integer>	any_level

%left	OR_P
%left	AND_P
%right	NOT_P
%left	'+' '-'
%left	'*' '/' '%'
%left	UMINUS
%nonassoc '(' ')'

/* Grammar follows */
%%

result:
	mode expr_or_predicate			{ pathlex.(*lexer).setResult($1, $2) }
	;

expr_or_predicate:
	expr							{ $$ = $1 }
	| predicate						{ $$ = $1; pathlex.(*lexer).setPred() }
	;

mode:
	STRICT_P						{ $$ = false }
	| LAX_P							{ $$ = true }
	| /* EMPTY */					{ $$ = true }
	;

scalar_value:
	STRING_P						{ $$ = ast.NewString($1) }
	| NULL_P						{ $$ = ast.NewConst(ast.ConstNull) }
	| TRUE_P						{ $$ = ast.NewConst(ast.ConstTrue) }
	| FALSE_P						{ $$ = ast.NewConst(ast.ConstFalse) }
	| NUMERIC_P						{ $$ = ast.NewNumeric($1) }
	| INT_P							{ $$ = ast.NewInteger($1) }
	| VARIABLE_P					{ $$ = ast.NewVariable($1) }
	;

comp_op:
	EQUAL_P							{ $$ = ast.BinaryEqual }
	| NOTEQUAL_P					{ $$ = ast.BinaryNotEqual }
	| LESS_P						{ $$ = ast.BinaryLess }
	| GREATER_P						{ $$ = ast.BinaryGreater }
	| LESSEQUAL_P					{ $$ = ast.BinaryLessOrEqual }
	| GREATEREQUAL_P				{ $$ = ast.BinaryGreaterOrEqual }
	;

delimited_predicate:
	'(' predicate ')'				{ $$ = $2 }
	| EXISTS_P '(' expr ')'			{ $$ = ast.NewUnary(ast.UnaryExists, $3) }
	;

predicate:
	delimited_predicate				{ $$ = $1 }
	| expr comp_op expr				{ $$ = ast.NewBinary($2, $1, $3) }
	| predicate AND_P predicate		{ $$ = ast.NewBinary(ast.BinaryAnd, $1, $3) }
	| predicate OR_P predicate		{ $$ = ast.NewBinary(ast.BinaryOr, $1, $3) }
	| NOT_P delimited_predicate		{ $$ = ast.NewUnary(ast.UnaryNot, $2) }
	| '(' predicate ')' IS_P UNKNOWN_P
									{ $$ = ast.NewUnary(ast.UnaryIsUnknown, $2) }
	| expr STARTS_P WITH_P starts_with_initial
									{ $$ = ast.NewBinary(ast.BinaryStartsWith, $1, $4) }
	| expr LIKE_REGEX_P STRING_P
	{
		var err error
		$$, err = ast.NewRegex($1, $3, "")
		if err != nil {
			pathlex.Error(err.Error())
		}
	}
	| expr LIKE_REGEX_P STRING_P FLAG_P STRING_P
	{
		var err error
		$$, err = ast.NewRegex($1, $3, $5)
		if err != nil {
			pathlex.Error(err.Error())
		}
	}
	;

starts_with_initial:
	STRING_P						{ $$ = ast.NewString($1) }
	| VARIABLE_P					{ $$ = ast.NewVariable($1) }
	;

path_primary:
	scalar_value					{ $$ = $1 }
	| '$'							{ $$ = ast.NewConst(ast.ConstRoot) }
	| '@'							{ $$ = ast.NewConst(ast.ConstCurrent) }
	| LAST_P						{ $$ = ast.NewConst(ast.ConstLast) }
	;

accessor_expr:
	path_primary					{ $$ = []ast.Node{$1} }
	| '(' expr ')' accessor_op		{ $$ = []ast.Node{$2, $4} }
	| '(' predicate ')' accessor_op	{ $$ = []ast.Node{$2, $4} }
	| accessor_expr accessor_op		{ $$ = append($$, $2) }
	;

expr:
	accessor_expr					{ $$ = ast.LinkNodes($1) }
	| '(' expr ')'					{ $$ = $2 }
	| '+' expr %prec UMINUS			{ $$ = ast.NewUnaryOrNumber(ast.UnaryPlus, $2) }
	| '-' expr %prec UMINUS			{ $$ = ast.NewUnaryOrNumber(ast.UnaryMinus, $2) }
	| expr '+' expr					{ $$ = ast.NewBinary(ast.BinaryAdd, $1, $3) }
	| expr '-' expr					{ $$ = ast.NewBinary(ast.BinarySub, $1, $3) }
	| expr '*' expr					{ $$ = ast.NewBinary(ast.BinaryMul, $1, $3) }
	| expr '/' expr					{ $$ = ast.NewBinary(ast.BinaryDiv, $1, $3) }
	| expr '%' expr					{ $$ = ast.NewBinary(ast.BinaryMod, $1, $3) }
	;

index_elem:
	expr							{ $$ = ast.NewBinary(ast.BinarySubscript, $1, nil) }
	| expr TO_P expr				{ $$ = ast.NewBinary(ast.BinarySubscript, $1, $3) }
	;

index_list:
	index_elem						{ $$ = []ast.Node{$1} }
	| index_list ',' index_elem		{ $$ = append($$, $3) }
	;

array_accessor:
	'[' '*' ']'						{ $$ = ast.NewConst(ast.ConstAnyArray) }
	| '[' index_list ']'			{ $$ = ast.NewArrayIndex($2) }
	;

any_level:
	INT_P							{ $$, _ = strconv.Atoi($1) }
	| LAST_P						{ $$ = -1 }
	;

any_path:
	ANY_P							{ $$ = ast.NewAny(0, -1) }
	| ANY_P '{' any_level '}'		{ $$ = ast.NewAny($3, $3) }
	| ANY_P '{' any_level TO_P any_level '}'
									{ $$ = ast.NewAny($3, $5) }
	;

accessor_op:
	'.' key							{ $$ = $2 }
	| '.' '*'						{ $$ = ast.NewConst(ast.ConstAnyKey) }
	| array_accessor				{ $$ = $1 }
	| '.' any_path					{ $$ = $2 }
	| '.' method '(' ')'			{ $$ = $2 }
	| '?' '(' predicate ')'			{ $$ = ast.NewUnary(ast.UnaryFilter, $3) }
	| '.' DECIMAL_P '(' opt_csv_list ')'
		{
			switch len($4) {
			case 0:
				$$ = ast.NewBinary(ast.BinaryDecimal, nil, nil)
			case 1:
				$$ = ast.NewBinary(ast.BinaryDecimal, $4[0], nil)
			case 2:
				$$ = ast.NewBinary(ast.BinaryDecimal, $4[0], $4[1])
			default:
				panic("invalid input syntax: .decimal() can only have an optional precision[,scale]")
			}
		}
	| '.' DATE_P '(' ')' { $$ = ast.NewUnary(ast.UnaryDate, nil) }
	| '.' DATETIME_P '(' opt_datetime_template ')'
		{ $$ = ast.NewUnary(ast.UnaryDateTime, $4) }
	| '.' TIME_P '(' opt_datetime_precision ')'
		{ $$ = ast.NewUnary(ast.UnaryTime, $4) }
	| '.' TIME_TZ_P '(' opt_datetime_precision ')'
		{ $$ = ast.NewUnary(ast.UnaryTimeTZ, $4) }
	| '.' TIMESTAMP_P '(' opt_datetime_precision ')'
		{ $$ = ast.NewUnary(ast.UnaryTimestamp, $4) }
	| '.' TIMESTAMP_TZ_P '(' opt_datetime_precision ')'
		{ $$ = ast.NewUnary(ast.UnaryTimestampTZ, $4) }
	;

csv_elem:
	INT_P
		{ $$ = ast.NewInteger($1) }
	| '+' INT_P %prec UMINUS
		{ $$ = ast.NewUnaryOrNumber(ast.UnaryPlus, ast.NewInteger($2)) }
	| '-' INT_P %prec UMINUS
		{ $$ = ast.NewUnaryOrNumber(ast.UnaryMinus, ast.NewInteger($2)) }
	;

csv_list:
	csv_elem						{ $$ = []ast.Node{$1} }
	| csv_list ',' csv_elem			{ $$ = append($$, $3) }
	;

opt_csv_list:
	csv_list						{ $$ = $1 }
	| /* EMPTY */					{ $$ = nil }
	;

datetime_precision:
	INT_P							{ $$ = ast.NewInteger($1) }
	;

opt_datetime_precision:
	datetime_precision				{ $$ = $1 }
	| /* EMPTY */					{ $$ = nil }
	;

datetime_template:
	STRING_P						{ $$ = ast.NewString($1) }
	;

opt_datetime_template:
	datetime_template				{ $$ = $1 }
	| /* EMPTY */					{ $$ = nil }
	;

key:
	key_name						{ $$ = ast.NewKey($1) }
	;

key_name:
	IDENT_P
	| STRING_P
	| TO_P
	| NULL_P
	| TRUE_P
	| FALSE_P
	| IS_P
	| UNKNOWN_P
	| EXISTS_P
	| STRICT_P
	| LAX_P
	| ABS_P
	| SIZE_P
	| TYPE_P
	| FLOOR_P
	| DOUBLE_P
	| CEILING_P
	| DATETIME_P
	| KEYVALUE_P
	| LAST_P
	| STARTS_P
	| WITH_P
	| LIKE_REGEX_P
	| FLAG_P
	| BIGINT_P
	| BOOLEAN_P
	| DATE_P
	| DECIMAL_P
	| INTEGER_P
	| NUMBER_P
	| STRINGFUNC_P
	| TIME_P
	| TIME_TZ_P
	| TIMESTAMP_P
	| TIMESTAMP_TZ_P
	;

method:
	ABS_P							{ $$ = ast.NewMethod(ast.MethodAbs) }
	| SIZE_P						{ $$ = ast.NewMethod(ast.MethodSize) }
	| TYPE_P						{ $$ = ast.NewMethod(ast.MethodType) }
	| FLOOR_P						{ $$ = ast.NewMethod(ast.MethodFloor) }
	| DOUBLE_P						{ $$ = ast.NewMethod(ast.MethodDouble) }
	| CEILING_P						{ $$ = ast.NewMethod(ast.MethodCeiling) }
	| KEYVALUE_P					{ $$ = ast.NewMethod(ast.MethodKeyValue) }
	| BIGINT_P						{ $$ = ast.NewMethod(ast.MethodBigInt) }
	| BOOLEAN_P						{ $$ = ast.NewMethod(ast.MethodBoolean) }
	| INTEGER_P						{ $$ = ast.NewMethod(ast.MethodInteger) }
	| NUMBER_P						{ $$ = ast.NewMethod(ast.MethodNumber) }
	| STRINGFUNC_P					{ $$ = ast.NewMethod(ast.MethodString) }
	;
%%
