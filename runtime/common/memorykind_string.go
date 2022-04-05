// Code generated by "stringer -type=MemoryKind -trimprefix=MemoryKind"; DO NOT EDIT.

package common

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[MemoryKindUnknown-0]
	_ = x[MemoryKindBool-1]
	_ = x[MemoryKindAddress-2]
	_ = x[MemoryKindString-3]
	_ = x[MemoryKindCharacter-4]
	_ = x[MemoryKindMetaType-5]
	_ = x[MemoryKindNumber-6]
	_ = x[MemoryKindArray-7]
	_ = x[MemoryKindDictionary-8]
	_ = x[MemoryKindComposite-9]
	_ = x[MemoryKindOptional-10]
	_ = x[MemoryKindNil-11]
	_ = x[MemoryKindVoid-12]
	_ = x[MemoryKindTypeValue-13]
	_ = x[MemoryKindPathValue-14]
	_ = x[MemoryKindCapabilityValue-15]
	_ = x[MemoryKindLinkValue-16]
	_ = x[MemoryKindStorageReferenceValue-17]
	_ = x[MemoryKindEphemeralReferenceValue-18]
	_ = x[MemoryKindInterpretedFunction-19]
	_ = x[MemoryKindHostFunction-20]
	_ = x[MemoryKindBoundFunction-21]
	_ = x[MemoryKindBigInt-22]
	_ = x[MemoryKindRawString-23]
	_ = x[MemoryKindAddressLocation-24]
	_ = x[MemoryKindBytes-25]
	_ = x[MemoryKindVariable-26]
	_ = x[MemoryKindTokenIdentifier-27]
	_ = x[MemoryKindTokenComment-28]
	_ = x[MemoryKindTokenNumericLiteral-29]
	_ = x[MemoryKindTokenSyntax-30]
	_ = x[MemoryKindIdentifier-31]
	_ = x[MemoryKindArgument-32]
	_ = x[MemoryKindBlock-33]
	_ = x[MemoryKindFunctionDeclaration-34]
	_ = x[MemoryKindCompositeDeclaration-35]
	_ = x[MemoryKindInterfaceDeclaration-36]
	_ = x[MemoryKindEnumCaseDeclaration-37]
	_ = x[MemoryKindFieldDeclaration-38]
	_ = x[MemoryKindTransactionDeclaration-39]
	_ = x[MemoryKindImportDeclaration-40]
	_ = x[MemoryKindVariableDeclaration-41]
	_ = x[MemoryKindSpecialFunctionDeclaration-42]
	_ = x[MemoryKindPragmaDeclaration-43]
}

const _MemoryKind_name = "UnknownBoolAddressStringCharacterMetaTypeNumberArrayDictionaryCompositeOptionalNilVoidTypeValuePathValueCapabilityValueLinkValueStorageReferenceValueEphemeralReferenceValueInterpretedFunctionHostFunctionBoundFunctionBigIntRawStringAddressLocationBytesVariableTokenIdentifierTokenCommentTokenNumericLiteralTokenSyntaxIdentifierArgumentBlockFunctionDeclarationCompositeDeclarationInterfaceDeclarationEnumCaseDeclarationFieldDeclarationTransactionDeclarationImportDeclarationVariableDeclarationSpecialFunctionDeclarationPragmaDeclaration"

var _MemoryKind_index = [...]uint16{0, 7, 11, 18, 24, 33, 41, 47, 52, 62, 71, 79, 82, 86, 95, 104, 119, 128, 149, 172, 191, 203, 216, 222, 231, 246, 251, 259, 274, 286, 305, 316, 326, 334, 339, 358, 378, 398, 417, 433, 455, 472, 491, 517, 534}

func (i MemoryKind) String() string {
	if i >= MemoryKind(len(_MemoryKind_index)-1) {
		return "MemoryKind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _MemoryKind_name[_MemoryKind_index[i]:_MemoryKind_index[i+1]]
}
