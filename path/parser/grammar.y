%{
# // !codeanalysis
/*-------------------------------------------------------------------------
 *
 * grammar.y
 *	 Grammar definitions for jsonpath datatype
 *
 * Transforms tokenized jsonpath into tree of JsonPathParseItem structs.
 *
 * Copyright (c) 2019-2024, PostgreSQL Global Development Group
 *
 * https://github.com/postgres/postgres/blob/b4a71cf/src/backend/utils/adt/jsonpath_gram.y
 *
 *-------------------------------------------------------------------------
 */

package parser

%}

%union{
	str string
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

/* Grammar follows */
%%

result: {  }
%%
