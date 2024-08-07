// Code generated by "stringer -linecomment -output ast_string.go -type Constant,BinaryOperator,UnaryOperator,MethodName"; DO NOT EDIT.

package ast

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ConstRoot-0]
	_ = x[ConstCurrent-1]
	_ = x[ConstLast-2]
	_ = x[ConstAnyArray-3]
	_ = x[ConstAnyKey-4]
	_ = x[ConstTrue-5]
	_ = x[ConstFalse-6]
	_ = x[ConstNull-7]
}

const _Constant_name = "$@last[*]*truefalsenull"

var _Constant_index = [...]uint8{0, 1, 2, 6, 9, 10, 14, 19, 23}

func (i Constant) String() string {
	if i < 0 || i >= Constant(len(_Constant_index)-1) {
		return "Constant(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Constant_name[_Constant_index[i]:_Constant_index[i+1]]
}
func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[BinaryAnd-0]
	_ = x[BinaryOr-1]
	_ = x[BinaryEqual-2]
	_ = x[BinaryNotEqual-3]
	_ = x[BinaryLess-4]
	_ = x[BinaryGreater-5]
	_ = x[BinaryLessOrEqual-6]
	_ = x[BinaryGreaterOrEqual-7]
	_ = x[BinaryStartsWith-8]
	_ = x[BinaryAdd-9]
	_ = x[BinarySub-10]
	_ = x[BinaryMul-11]
	_ = x[BinaryDiv-12]
	_ = x[BinaryMod-13]
	_ = x[BinarySubscript-14]
	_ = x[BinaryDecimal-15]
}

const _BinaryOperator_name = "&&||==!=<><=>=starts with+-*/%to.decimal()"

var _BinaryOperator_index = [...]uint8{0, 2, 4, 6, 8, 9, 10, 12, 14, 25, 26, 27, 28, 29, 30, 32, 42}

func (i BinaryOperator) String() string {
	if i < 0 || i >= BinaryOperator(len(_BinaryOperator_index)-1) {
		return "BinaryOperator(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _BinaryOperator_name[_BinaryOperator_index[i]:_BinaryOperator_index[i+1]]
}
func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[UnaryExists-0]
	_ = x[UnaryNot-1]
	_ = x[UnaryIsUnknown-2]
	_ = x[UnaryPlus-3]
	_ = x[UnaryMinus-4]
	_ = x[UnaryFilter-5]
	_ = x[UnaryDateTime-6]
	_ = x[UnaryDate-7]
	_ = x[UnaryTime-8]
	_ = x[UnaryTimeTZ-9]
	_ = x[UnaryTimestamp-10]
	_ = x[UnaryTimestampTZ-11]
}

const _UnaryOperator_name = "exists!is unknown+-?.datetime.date.time.time_tz.timestamp.timestamp_tz"

var _UnaryOperator_index = [...]uint8{0, 6, 7, 17, 18, 19, 20, 29, 34, 39, 47, 57, 70}

func (i UnaryOperator) String() string {
	if i < 0 || i >= UnaryOperator(len(_UnaryOperator_index)-1) {
		return "UnaryOperator(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _UnaryOperator_name[_UnaryOperator_index[i]:_UnaryOperator_index[i+1]]
}
func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[MethodAbs-0]
	_ = x[MethodSize-1]
	_ = x[MethodType-2]
	_ = x[MethodFloor-3]
	_ = x[MethodCeiling-4]
	_ = x[MethodDouble-5]
	_ = x[MethodKeyValue-6]
	_ = x[MethodBigInt-7]
	_ = x[MethodBoolean-8]
	_ = x[MethodInteger-9]
	_ = x[MethodNumber-10]
	_ = x[MethodString-11]
}

const _MethodName_name = ".abs().size().type().floor().ceiling().double().keyvalue().bigint().boolean().integer().number().string()"

var _MethodName_index = [...]uint8{0, 6, 13, 20, 28, 38, 47, 58, 67, 77, 87, 96, 105}

func (i MethodName) String() string {
	if i < 0 || i >= MethodName(len(_MethodName_index)-1) {
		return "MethodName(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _MethodName_name[_MethodName_index[i]:_MethodName_index[i+1]]
}
