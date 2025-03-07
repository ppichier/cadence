/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

import (
	"math/big"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

// exportValue converts a runtime value to its native Go representation.
func exportValue(
	value exportableValue,
	getLocationRange func() interpreter.LocationRange,
) (
	cadence.Value,
	error,
) {
	return exportValueWithInterpreter(
		value.Value,
		value.Interpreter(),
		getLocationRange,
		seenReferences{},
	)
}

// ExportValue converts a runtime value to its native Go representation.
func ExportValue(
	value interpreter.Value,
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
) (cadence.Value, error) {
	return exportValueWithInterpreter(
		value,
		inter,
		getLocationRange,
		seenReferences{},
	)
}

// NOTE: Do not generalize to map[interpreter.Value],
// as not all values are Go hashable, i.e. this might lead to run-time panics
type seenReferences map[*interpreter.EphemeralReferenceValue]struct{}

// exportValueWithInterpreter exports the given internal (interpreter) value to an external value.
//
// The export is recursive, the results parameter prevents cycles:
// it is checked at the start of the recursively called function,
// and pre-set before a recursive call.
//
func exportValueWithInterpreter(
	value interpreter.Value,
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Value,
	error,
) {

	switch v := value.(type) {
	case interpreter.VoidValue:
		return cadence.NewMeteredVoid(inter), nil
	case interpreter.NilValue:
		return cadence.NewMeteredOptional(inter, nil), nil
	case *interpreter.SomeValue:
		return exportSomeValue(v, inter, getLocationRange, seenReferences)
	case interpreter.BoolValue:
		return cadence.NewMeteredBool(inter, bool(v)), nil
	case *interpreter.StringValue:
		return cadence.NewMeteredString(
			inter,
			common.NewCadenceStringMemoryUsage(len(v.Str)),
			func() string {
				return v.Str
			},
		)
	case interpreter.CharacterValue:
		return cadence.NewMeteredCharacter(
			inter,
			common.NewCadenceCharacterMemoryUsage(len(v)),
			func() string {
				return string(v)
			},
		)
	case *interpreter.ArrayValue:
		return exportArrayValue(
			v,
			inter,
			getLocationRange,
			seenReferences,
		)
	case interpreter.IntValue:
		bigInt := v.ToBigInt(inter)
		return cadence.NewMeteredIntFromBig(
			inter,
			common.NewCadenceIntMemoryUsage(
				common.BigIntByteLength(bigInt),
			),
			func() *big.Int {
				return bigInt
			},
		), nil
	case interpreter.Int8Value:
		return cadence.NewMeteredInt8(inter, int8(v)), nil
	case interpreter.Int16Value:
		return cadence.NewMeteredInt16(inter, int16(v)), nil
	case interpreter.Int32Value:
		return cadence.NewMeteredInt32(inter, int32(v)), nil
	case interpreter.Int64Value:
		return cadence.NewMeteredInt64(inter, int64(v)), nil
	case interpreter.Int128Value:
		return cadence.NewMeteredInt128FromBig(
			inter,
			func() *big.Int {
				return v.ToBigInt(inter)
			},
		)
	case interpreter.Int256Value:
		return cadence.NewMeteredInt256FromBig(
			inter,
			func() *big.Int {
				return v.ToBigInt(inter)
			},
		)
	case interpreter.UIntValue:
		bigInt := v.ToBigInt(inter)
		return cadence.NewMeteredUIntFromBig(
			inter,
			common.NewCadenceIntMemoryUsage(
				common.BigIntByteLength(bigInt),
			),
			func() *big.Int {
				return bigInt
			},
		)
	case interpreter.UInt8Value:
		return cadence.NewMeteredUInt8(inter, uint8(v)), nil
	case interpreter.UInt16Value:
		return cadence.NewMeteredUInt16(inter, uint16(v)), nil
	case interpreter.UInt32Value:
		return cadence.NewMeteredUInt32(inter, uint32(v)), nil
	case interpreter.UInt64Value:
		return cadence.NewMeteredUInt64(inter, uint64(v)), nil
	case interpreter.UInt128Value:
		return cadence.NewMeteredUInt128FromBig(
			inter,
			func() *big.Int {
				return v.ToBigInt(inter)
			},
		)
	case interpreter.UInt256Value:
		return cadence.NewMeteredUInt256FromBig(
			inter,
			func() *big.Int {
				return v.ToBigInt(inter)
			},
		)
	case interpreter.Word8Value:
		return cadence.NewMeteredWord8(inter, uint8(v)), nil
	case interpreter.Word16Value:
		return cadence.NewMeteredWord16(inter, uint16(v)), nil
	case interpreter.Word32Value:
		return cadence.NewMeteredWord32(inter, uint32(v)), nil
	case interpreter.Word64Value:
		return cadence.NewMeteredWord64(inter, uint64(v)), nil
	case interpreter.Fix64Value:
		return cadence.Fix64(v), nil
	case interpreter.UFix64Value:
		return cadence.UFix64(v), nil
	case *interpreter.CompositeValue:
		return exportCompositeValue(
			v,
			inter,
			getLocationRange,
			seenReferences,
		)
	case *interpreter.SimpleCompositeValue:
		return exportSimpleCompositeValue(
			v,
			inter,
			getLocationRange,
			seenReferences,
		)
	case *interpreter.DictionaryValue:
		return exportDictionaryValue(
			v,
			inter,
			getLocationRange,
			seenReferences,
		)
	case interpreter.AddressValue:
		return cadence.NewMeteredAddress(inter, v), nil
	case interpreter.LinkValue:
		return exportLinkValue(v, inter), nil
	case interpreter.PathValue:
		return exportPathValue(inter, v), nil
	case interpreter.TypeValue:
		return exportTypeValue(v, inter), nil
	case *interpreter.CapabilityValue:
		return exportCapabilityValue(v, inter), nil
	case *interpreter.EphemeralReferenceValue:
		// Break recursion through ephemeral references
		if _, ok := seenReferences[v]; ok {
			return nil, nil
		}
		defer delete(seenReferences, v)
		seenReferences[v] = struct{}{}
		return exportValueWithInterpreter(
			v.Value,
			inter,
			getLocationRange,
			seenReferences,
		)
	case *interpreter.StorageReferenceValue:
		referencedValue := v.ReferencedValue(inter)
		if referencedValue == nil {
			return nil, nil
		}
		return exportValueWithInterpreter(
			*referencedValue,
			inter,
			getLocationRange,
			seenReferences,
		)
	default:
		return nil, errors.NewUnexpectedError("cannot export value of type %T", value)
	}
}

func exportSomeValue(
	v *interpreter.SomeValue,
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Optional,
	error,
) {
	innerValue := v.InnerValue(inter, getLocationRange)

	if innerValue == nil {
		return cadence.NewMeteredOptional(inter, nil), nil
	}

	value, err := exportValueWithInterpreter(
		innerValue,
		inter,
		getLocationRange,
		seenReferences,
	)
	if err != nil {
		return cadence.Optional{}, err
	}

	return cadence.NewMeteredOptional(inter, value), nil
}

func exportArrayValue(
	v *interpreter.ArrayValue,
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Array,
	error,
) {
	array, err := cadence.NewMeteredArray(
		inter,
		v.Count(),
		func() ([]cadence.Value, error) {
			values := make([]cadence.Value, 0, v.Count())

			var err error
			v.Iterate(inter, func(value interpreter.Value) (resume bool) {
				var exportedValue cadence.Value
				exportedValue, err = exportValueWithInterpreter(
					value,
					inter,
					getLocationRange,
					seenReferences,
				)
				if err != nil {
					return false
				}
				values = append(
					values,
					exportedValue,
				)
				return true
			})

			if err != nil {
				return nil, err
			}
			return values, nil
		},
	)
	if err != nil {
		return cadence.Array{}, err
	}

	exportType := ExportType(v.SemaType(inter), map[sema.TypeID]cadence.Type{}).(cadence.ArrayType)

	return array.WithType(exportType), err
}

func exportCompositeValue(
	v *interpreter.CompositeValue,
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Value,
	error,
) {

	staticType, err := inter.ConvertStaticToSemaType(v.StaticType(inter))
	if err != nil {
		return nil, err
	}

	compositeType, ok := staticType.(*sema.CompositeType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// TODO: consider making the results map "global", by moving it up to exportValueWithInterpreter
	t := exportCompositeType(inter, compositeType, map[sema.TypeID]cadence.Type{})

	// NOTE: use the exported type's fields to ensure fields in type
	// and value are in sync

	fieldNames := t.CompositeFields()

	makeFields := func() ([]cadence.Value, error) {
		fields := make([]cadence.Value, len(fieldNames))

		for i, field := range fieldNames {
			fieldName := field.Identifier

			fieldValue := v.GetField(inter, getLocationRange, fieldName)
			if fieldValue == nil && v.ComputedFields != nil {
				if computedField, ok := v.ComputedFields[fieldName]; ok {
					fieldValue = computedField(inter, getLocationRange)
				}
			}

			exportedFieldValue, err := exportValueWithInterpreter(
				fieldValue,
				inter,
				getLocationRange,
				seenReferences,
			)
			if err != nil {
				return nil, err
			}
			fields[i] = exportedFieldValue
		}

		return fields, nil
	}

	// NOTE: when modifying the cases below,
	// also update the error message below!

	switch compositeType.Kind {
	case common.CompositeKindStructure:
		structure, err := cadence.NewMeteredStruct(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return structure.WithType(t.(*cadence.StructType)), nil
	case common.CompositeKindResource:
		resource, err := cadence.NewMeteredResource(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return resource.WithType(t.(*cadence.ResourceType)), nil
	case common.CompositeKindEvent:
		event, err := cadence.NewMeteredEvent(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return event.WithType(t.(*cadence.EventType)), nil
	case common.CompositeKindContract:
		contract, err := cadence.NewMeteredContract(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return contract.WithType(t.(*cadence.ContractType)), nil
	case common.CompositeKindEnum:
		enum, err := cadence.NewMeteredEnum(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return enum.WithType(t.(*cadence.EnumType)), nil
	}

	return nil, errors.NewDefaultUserError(
		"invalid composite kind `%s`, must be %s",
		compositeType.Kind,
		common.EnumerateWords(
			[]string{
				common.CompositeKindStructure.Name(),
				common.CompositeKindResource.Name(),
				common.CompositeKindEvent.Name(),
				common.CompositeKindContract.Name(),
				common.CompositeKindEnum.Name(),
			},
			"or",
		),
	)
}

func exportSimpleCompositeValue(
	v *interpreter.SimpleCompositeValue,
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Value,
	error,
) {
	staticType, err := inter.ConvertStaticToSemaType(v.StaticType(inter))
	if err != nil {
		return nil, err
	}

	compositeType, ok := staticType.(*sema.CompositeType)
	if !ok {
		return nil, errors.NewUnexpectedError(
			"unexportable composite value: %s",
			staticType,
		)
	}

	// TODO: consider making the results map "global", by moving it up to exportValueWithInterpreter
	t := exportCompositeType(inter, compositeType, map[sema.TypeID]cadence.Type{})

	// NOTE: use the exported type's fields to ensure fields in type
	// and value are in sync

	fieldNames := t.CompositeFields()

	makeFields := func() ([]cadence.Value, error) {
		fields := make([]cadence.Value, len(fieldNames))

		for i, field := range fieldNames {
			fieldName := field.Identifier

			fieldValue := v.Fields[fieldName]
			if fieldValue == nil && v.ComputedFields != nil {
				if computedField, ok := v.ComputedFields[fieldName]; ok {
					fieldValue = computedField(inter, getLocationRange)
				}
			}

			exportedFieldValue, err := exportValueWithInterpreter(
				fieldValue,
				inter,
				getLocationRange,
				seenReferences,
			)
			if err != nil {
				return nil, err
			}
			fields[i] = exportedFieldValue
		}

		return fields, nil
	}

	// NOTE: when modifying the cases below,
	// also update the error message below!

	switch compositeType.Kind {
	case common.CompositeKindStructure:
		structure, err := cadence.NewMeteredStruct(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return structure.WithType(t.(*cadence.StructType)), nil
	case common.CompositeKindResource:
		resource, err := cadence.NewMeteredResource(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return resource.WithType(t.(*cadence.ResourceType)), nil
	case common.CompositeKindEvent:
		event, err := cadence.NewMeteredEvent(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return event.WithType(t.(*cadence.EventType)), nil
	case common.CompositeKindContract:
		contract, err := cadence.NewMeteredContract(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return contract.WithType(t.(*cadence.ContractType)), nil
	case common.CompositeKindEnum:
		enum, err := cadence.NewMeteredEnum(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return enum.WithType(t.(*cadence.EnumType)), nil
	}

	return nil, errors.NewUnexpectedError(
		"invalid composite kind `%s`, must be %s",
		compositeType.Kind,
		common.EnumerateWords(
			[]string{
				common.CompositeKindStructure.Name(),
				common.CompositeKindResource.Name(),
				common.CompositeKindEvent.Name(),
				common.CompositeKindContract.Name(),
				common.CompositeKindEnum.Name(),
			},
			"or",
		),
	)
}

func exportDictionaryValue(
	v *interpreter.DictionaryValue,
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Dictionary,
	error,
) {
	dictionary, err := cadence.NewMeteredDictionary(
		inter,
		v.Count(),
		func() ([]cadence.KeyValuePair, error) {
			var err error
			pairs := make([]cadence.KeyValuePair, 0, v.Count())

			v.Iterate(inter, func(key, value interpreter.Value) (resume bool) {

				var convertedKey cadence.Value
				convertedKey, err = exportValueWithInterpreter(
					key,
					inter,
					getLocationRange,
					seenReferences,
				)
				if err != nil {
					return false
				}

				var convertedValue cadence.Value
				convertedValue, err = exportValueWithInterpreter(
					value,
					inter,
					getLocationRange,
					seenReferences,
				)
				if err != nil {
					return false
				}

				pairs = append(
					pairs,
					cadence.KeyValuePair{
						Key:   convertedKey,
						Value: convertedValue,
					},
				)

				return true
			})

			if err != nil {
				return nil, err
			}

			return pairs, nil
		},
	)
	if err != nil {
		return cadence.Dictionary{}, err
	}

	exportType := ExportType(v.SemaType(inter), map[sema.TypeID]cadence.Type{}).(cadence.DictionaryType)

	return dictionary.WithType(exportType), err
}

func exportLinkValue(v interpreter.LinkValue, inter *interpreter.Interpreter) cadence.Link {
	path := exportPathValue(inter, v.TargetPath)
	ty := string(inter.MustConvertStaticToSemaType(v.Type).ID())
	return cadence.NewMeteredLink(inter, path, ty)
}

func exportPathValue(gauge common.MemoryGauge, v interpreter.PathValue) cadence.Path {
	domain := v.Domain.Identifier()
	common.UseMemory(gauge, common.MemoryUsage{
		Kind: common.MemoryKindRawString,
		// no need to add 1 to account for empty string: string is metered in Path struct
		Amount: uint64(len(domain)),
	})

	return cadence.NewMeteredPath(
		gauge,
		domain,
		v.Identifier,
	)
}

func exportTypeValue(v interpreter.TypeValue, inter *interpreter.Interpreter) cadence.TypeValue {
	var typ sema.Type
	if v.Type != nil {
		typ = inter.MustConvertStaticToSemaType(v.Type)
	}
	return cadence.NewMeteredTypeValue(
		inter,
		ExportMeteredType(inter, typ, map[sema.TypeID]cadence.Type{}),
	)
}

func exportCapabilityValue(v *interpreter.CapabilityValue, inter *interpreter.Interpreter) cadence.Capability {
	var borrowType sema.Type
	if v.BorrowType != nil {
		borrowType = inter.MustConvertStaticToSemaType(v.BorrowType)
	}

	return cadence.NewMeteredCapability(
		inter,
		exportPathValue(inter, v.Path),
		cadence.NewMeteredAddress(inter, v.Address),
		ExportMeteredType(inter, borrowType, map[sema.TypeID]cadence.Type{}),
	)
}

// exportEvent converts a runtime event to its native Go representation.
func exportEvent(
	gauge common.MemoryGauge,
	event exportableEvent,
	getLocationRange func() interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Event,
	error,
) {
	exported, err := cadence.NewMeteredEvent(
		gauge,
		len(event.Fields),
		func() ([]cadence.Value, error) {
			fields := make([]cadence.Value, len(event.Fields))

			for i, field := range event.Fields {
				value, err := exportValueWithInterpreter(
					field.Value,
					field.Interpreter(),
					getLocationRange,
					seenReferences,
				)
				if err != nil {
					return nil, err
				}
				fields[i] = value
			}

			return fields, nil
		},
	)

	if err != nil {
		return cadence.Event{}, err
	}

	eventType := ExportMeteredType(gauge, event.Type, map[sema.TypeID]cadence.Type{}).(*cadence.EventType)

	return exported.WithType(eventType), nil
}

// importValue converts a Cadence value to a runtime value.
func importValue(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	value cadence.Value,
	expectedType sema.Type,
) (interpreter.Value, error) {
	switch v := value.(type) {
	case cadence.Void:
		return interpreter.NewVoidValue(inter), nil
	case cadence.Optional:
		return importOptionalValue(
			inter,
			getLocationRange,
			v,
			expectedType,
		)
	case cadence.Bool:
		return interpreter.NewBoolValue(inter, bool(v)), nil
	case cadence.String:
		return importString(inter, v), nil
	case cadence.Character:
		return importCharacter(inter, v), nil
	case cadence.Bytes:
		return interpreter.ByteSliceToByteArrayValue(inter, v), nil
	case cadence.Address:
		return importAddress(inter, v), nil
	case cadence.Int:
		return importInt(inter, v), nil
	case cadence.Int8:
		return importInt8(inter, v), nil
	case cadence.Int16:
		return importInt16(inter, v), nil
	case cadence.Int32:
		return importInt32(inter, v), nil
	case cadence.Int64:
		return importInt64(inter, v), nil
	case cadence.Int128:
		return importInt128(inter, v), nil
	case cadence.Int256:
		return importInt256(inter, v), nil
	case cadence.UInt:
		return importUInt(inter, v), nil
	case cadence.UInt8:
		return importUInt8(inter, v), nil
	case cadence.UInt16:
		return importUInt16(inter, v), nil
	case cadence.UInt32:
		return importUInt32(inter, v), nil
	case cadence.UInt64:
		return importUInt64(inter, v), nil
	case cadence.UInt128:
		return importUInt128(inter, v), nil
	case cadence.UInt256:
		return importUInt256(inter, v), nil
	case cadence.Word8:
		return importWord8(inter, v), nil
	case cadence.Word16:
		return importWord16(inter, v), nil
	case cadence.Word32:
		return importWord32(inter, v), nil
	case cadence.Word64:
		return importWord64(inter, v), nil
	case cadence.Fix64:
		return importFix64(inter, v), nil
	case cadence.UFix64:
		return importUFix64(inter, v), nil
	case cadence.Path:
		return importPathValue(inter, v), nil
	case cadence.Array:
		return importArrayValue(
			inter,
			getLocationRange,
			v,
			expectedType,
		)
	case cadence.Dictionary:
		return importDictionaryValue(
			inter,
			getLocationRange,
			v,
			expectedType,
		)
	case cadence.Struct:
		return importCompositeValue(
			inter,
			getLocationRange,
			common.CompositeKindStructure,
			v.StructType.Location,
			v.StructType.QualifiedIdentifier,
			v.StructType.Fields,
			v.Fields,
		)
	case cadence.Resource:
		return importCompositeValue(
			inter,
			getLocationRange,
			common.CompositeKindResource,
			v.ResourceType.Location,
			v.ResourceType.QualifiedIdentifier,
			v.ResourceType.Fields,
			v.Fields,
		)
	case cadence.Event:
		return importCompositeValue(
			inter,
			getLocationRange,
			common.CompositeKindEvent,
			v.EventType.Location,
			v.EventType.QualifiedIdentifier,
			v.EventType.Fields,
			v.Fields,
		)
	case cadence.Enum:
		return importCompositeValue(
			inter,
			getLocationRange,
			common.CompositeKindEnum,
			v.EnumType.Location,
			v.EnumType.QualifiedIdentifier,
			v.EnumType.Fields,
			v.Fields,
		)
	case cadence.TypeValue:
		return importTypeValue(
			inter,
			v.StaticType,
		)
	case cadence.Capability:
		return importCapability(
			inter,
			v.Path,
			v.Address,
			v.BorrowType,
		)
	default:
		// This means the implementation has unhandled types.
		// Hence, return an internal error
		return nil, errors.NewUnexpectedError("cannot import value of type %T", value)
	}
}
func importUInt8(inter *interpreter.Interpreter, v cadence.UInt8) interpreter.UInt8Value {
	return interpreter.NewUInt8Value(
		inter,
		func() uint8 {
			return uint8(v)
		},
	)
}

func importUInt16(inter *interpreter.Interpreter, v cadence.UInt16) interpreter.UInt16Value {
	return interpreter.NewUInt16Value(
		inter,
		func() uint16 {
			return uint16(v)
		},
	)
}

func importUInt32(inter *interpreter.Interpreter, v cadence.UInt32) interpreter.UInt32Value {
	return interpreter.NewUInt32Value(
		inter,
		func() uint32 {
			return uint32(v)
		},
	)
}

func importUInt64(inter *interpreter.Interpreter, v cadence.UInt64) interpreter.UInt64Value {
	return interpreter.NewUInt64Value(
		inter,
		func() uint64 {
			return uint64(v)
		},
	)
}

func importUInt128(inter *interpreter.Interpreter, v cadence.UInt128) interpreter.UInt128Value {
	return interpreter.NewUInt128ValueFromBigInt(
		inter,
		func() *big.Int {
			return v.Value
		},
	)
}

func importUInt256(inter *interpreter.Interpreter, v cadence.UInt256) interpreter.UInt256Value {
	return interpreter.NewUInt256ValueFromBigInt(
		inter,
		func() *big.Int {
			return v.Value
		},
	)
}

func importInt(inter *interpreter.Interpreter, v cadence.Int) interpreter.IntValue {
	memoryUsage := common.NewBigIntMemoryUsage(
		common.BigIntByteLength(v.Value),
	)
	return interpreter.NewIntValueFromBigInt(
		inter,
		memoryUsage,
		func() *big.Int {
			return v.Value
		},
	)
}

func importInt8(inter *interpreter.Interpreter, v cadence.Int8) interpreter.Int8Value {
	return interpreter.NewInt8Value(
		inter,
		func() int8 {
			return int8(v)
		},
	)
}

func importInt16(inter *interpreter.Interpreter, v cadence.Int16) interpreter.Int16Value {
	return interpreter.NewInt16Value(
		inter,
		func() int16 {
			return int16(v)
		},
	)
}

func importInt32(inter *interpreter.Interpreter, v cadence.Int32) interpreter.Int32Value {
	return interpreter.NewInt32Value(
		inter,
		func() int32 {
			return int32(v)
		},
	)
}

func importInt64(inter *interpreter.Interpreter, v cadence.Int64) interpreter.Int64Value {
	return interpreter.NewInt64Value(
		inter,
		func() int64 {
			return int64(v)
		},
	)
}

func importInt128(inter *interpreter.Interpreter, v cadence.Int128) interpreter.Int128Value {
	return interpreter.NewInt128ValueFromBigInt(
		inter,
		func() *big.Int {
			return v.Value
		},
	)
}

func importInt256(inter *interpreter.Interpreter, v cadence.Int256) interpreter.Int256Value {
	return interpreter.NewInt256ValueFromBigInt(
		inter,
		func() *big.Int {
			return v.Value
		},
	)
}

func importUInt(inter *interpreter.Interpreter, v cadence.UInt) interpreter.UIntValue {
	memoryUsage := common.NewBigIntMemoryUsage(
		common.BigIntByteLength(v.Value),
	)
	return interpreter.NewUIntValueFromBigInt(
		inter,
		memoryUsage,
		func() *big.Int {
			return v.Value
		},
	)
}

func importWord8(inter *interpreter.Interpreter, v cadence.Word8) interpreter.Word8Value {
	return interpreter.NewWord8Value(
		inter,
		func() uint8 {
			return uint8(v)
		},
	)
}

func importWord16(inter *interpreter.Interpreter, v cadence.Word16) interpreter.Word16Value {
	return interpreter.NewWord16Value(
		inter,
		func() uint16 {
			return uint16(v)
		},
	)
}

func importWord32(inter *interpreter.Interpreter, v cadence.Word32) interpreter.Word32Value {
	return interpreter.NewWord32Value(
		inter,
		func() uint32 {
			return uint32(v)
		},
	)
}

func importWord64(inter *interpreter.Interpreter, v cadence.Word64) interpreter.Word64Value {
	return interpreter.NewWord64Value(
		inter,
		func() uint64 {
			return uint64(v)
		},
	)
}

func importFix64(inter *interpreter.Interpreter, v cadence.Fix64) interpreter.Fix64Value {
	return interpreter.NewFix64Value(
		inter,
		func() int64 {
			return int64(v)
		},
	)
}

func importUFix64(inter *interpreter.Interpreter, v cadence.UFix64) interpreter.UFix64Value {
	return interpreter.NewUFix64Value(
		inter,
		func() uint64 {
			return uint64(v)
		},
	)
}

func importString(inter *interpreter.Interpreter, v cadence.String) *interpreter.StringValue {
	memoryUsage := common.NewStringMemoryUsage(len(v))
	return interpreter.NewStringValue(
		inter,
		memoryUsage,
		func() string {
			return string(v)
		},
	)
}

func importCharacter(inter *interpreter.Interpreter, v cadence.Character) interpreter.CharacterValue {
	s := string(v)
	memoryUsage := common.NewCharacterMemoryUsage(len(s))
	return interpreter.NewCharacterValue(
		inter,
		memoryUsage,
		func() string {
			return s
		},
	)
}

func importAddress(inter *interpreter.Interpreter, v cadence.Address) interpreter.AddressValue {
	return interpreter.NewAddressValue(
		inter,
		common.Address(v),
	)
}

func importPathValue(inter *interpreter.Interpreter, v cadence.Path) interpreter.PathValue {
	// meter the Path's Identifier since path is just a container
	common.UseMemory(inter, common.NewRawStringMemoryUsage(len(v.Identifier)))

	return interpreter.NewPathValue(
		inter,
		common.PathDomainFromIdentifier(v.Domain),
		v.Identifier,
	)
}

func importTypeValue(
	inter *interpreter.Interpreter,
	v cadence.Type,
) (
	interpreter.TypeValue,
	error,
) {
	typ := ImportType(inter, v)
	/* creating a static type performs no validation, so
	   in order to be sure the type we have created is legal,
	   we convert it to a sema type. If this fails, the
	   import is invalid */
	_, err := inter.ConvertStaticToSemaType(typ)
	if err != nil {
		// unmetered because when err != nil, value should be ignored
		return interpreter.EmptyTypeValue, err
	}

	return interpreter.NewTypeValue(inter, typ), nil
}

func importCapability(
	inter *interpreter.Interpreter,
	path cadence.Path,
	address cadence.Address,
	borrowType cadence.Type,
) (
	*interpreter.CapabilityValue,
	error,
) {

	_, ok := borrowType.(cadence.ReferenceType)
	if !ok {
		return nil, errors.NewDefaultUserError(
			"cannot import capability: expected reference, got '%s'",
			borrowType.ID(),
		)
	}

	return interpreter.NewCapabilityValue(
		inter,
		interpreter.NewAddressValue(
			inter,
			common.Address(address),
		),
		importPathValue(inter, path),
		ImportType(inter, borrowType),
	), nil

}

func importOptionalValue(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	v cadence.Optional,
	expectedType sema.Type,
) (
	interpreter.Value,
	error,
) {
	if v.Value == nil {
		return interpreter.NewNilValue(inter), nil
	}

	var innerType sema.Type
	if optionalType, ok := expectedType.(*sema.OptionalType); ok {
		innerType = optionalType.Type
	}

	innerValue, err := importValue(inter, getLocationRange, v.Value, innerType)
	if err != nil {
		return nil, err
	}

	return interpreter.NewSomeValueNonCopying(inter, innerValue), nil
}

func importArrayValue(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	v cadence.Array,
	expectedType sema.Type,
) (
	*interpreter.ArrayValue,
	error,
) {
	values := make([]interpreter.Value, len(v.Values))

	var elementType sema.Type
	arrayType, ok := expectedType.(sema.ArrayType)
	if ok {
		elementType = arrayType.ElementType(false)
	}

	for i, element := range v.Values {
		value, err := importValue(
			inter,
			getLocationRange,
			element,
			elementType,
		)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}

	var staticArrayType interpreter.ArrayStaticType
	if arrayType != nil {
		staticArrayType = interpreter.ConvertSemaArrayTypeToStaticArrayType(inter, arrayType)
	} else {
		types := make([]sema.Type, len(v.Values))

		for i, value := range values {
			typ, err := inter.ConvertStaticToSemaType(value.StaticType(inter))
			if err != nil {
				return nil, err
			}
			types[i] = typ
		}

		elementSuperType := sema.LeastCommonSuperType(types...)
		if elementSuperType == sema.InvalidType {
			return nil, errors.NewUnexpectedError("cannot import array: elements do not belong to the same type")
		}

		staticArrayType = interpreter.NewVariableSizedStaticType(
			inter,
			interpreter.ConvertSemaToStaticType(inter, elementSuperType),
		)
	}

	return interpreter.NewArrayValue(
		inter,
		getLocationRange,
		staticArrayType,
		common.Address{},
		values...,
	), nil
}

func importDictionaryValue(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	v cadence.Dictionary,
	expectedType sema.Type,
) (
	*interpreter.DictionaryValue,
	error,
) {
	keysAndValues := make([]interpreter.Value, len(v.Pairs)*2)

	var keyType sema.Type
	var valueType sema.Type

	dictionaryType, ok := expectedType.(*sema.DictionaryType)
	if ok {
		keyType = dictionaryType.KeyType
		valueType = dictionaryType.ValueType
	}

	for i, pair := range v.Pairs {
		key, err := importValue(
			inter,
			getLocationRange,
			pair.Key,
			keyType,
		)
		if err != nil {
			return nil, err
		}
		keysAndValues[i*2] = key

		value, err := importValue(
			inter,
			getLocationRange,
			pair.Value,
			valueType,
		)
		if err != nil {
			return nil, err
		}
		keysAndValues[i*2+1] = value
	}

	var dictionaryStaticType interpreter.DictionaryStaticType
	if dictionaryType != nil {
		dictionaryStaticType = interpreter.ConvertSemaDictionaryTypeToStaticDictionaryType(inter, dictionaryType)
	} else {
		size := len(v.Pairs)
		keyTypes := make([]sema.Type, size)
		valueTypes := make([]sema.Type, size)

		for i := 0; i < size; i++ {
			keyType, err := inter.ConvertStaticToSemaType(keysAndValues[i*2].StaticType(inter))
			if err != nil {
				return nil, err
			}
			keyTypes[i] = keyType

			valueType, err := inter.ConvertStaticToSemaType(keysAndValues[i*2+1].StaticType(inter))
			if err != nil {
				return nil, err
			}
			valueTypes[i] = valueType
		}

		keySuperType := sema.LeastCommonSuperType(keyTypes...)
		valueSuperType := sema.LeastCommonSuperType(valueTypes...)

		if !sema.IsValidDictionaryKeyType(keySuperType) {
			return nil, errors.NewDefaultUserError(
				"cannot import dictionary: keys does not belong to the same type",
			)
		}

		if valueSuperType == sema.InvalidType {
			return nil, errors.NewDefaultUserError("cannot import dictionary: values does not belong to the same type")
		}

		dictionaryStaticType = interpreter.NewDictionaryStaticType(
			inter,
			interpreter.ConvertSemaToStaticType(inter, keySuperType),
			interpreter.ConvertSemaToStaticType(inter, valueSuperType),
		)
	}

	return interpreter.NewDictionaryValue(
		inter,
		getLocationRange,
		dictionaryStaticType,
		keysAndValues...,
	), nil
}

func importCompositeValue(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	kind common.CompositeKind,
	location Location,
	qualifiedIdentifier string,
	fieldTypes []cadence.Field,
	fieldValues []cadence.Value,
) (
	*interpreter.CompositeValue,
	error,
) {
	var fields []interpreter.CompositeField

	typeID := common.NewTypeIDFromQualifiedName(inter, location, qualifiedIdentifier)
	compositeType, typeErr := inter.GetCompositeType(location, qualifiedIdentifier, typeID)
	if typeErr != nil {
		return nil, typeErr
	}

	for i := 0; i < len(fieldTypes) && i < len(fieldValues); i++ {
		fieldType := fieldTypes[i]
		fieldValue := fieldValues[i]

		var expectedFieldType sema.Type

		member, ok := compositeType.Members.Get(fieldType.Identifier)
		if ok {
			expectedFieldType = member.TypeAnnotation.Type
		}

		importedFieldValue, err := importValue(
			inter,
			getLocationRange,
			fieldValue,
			expectedFieldType,
		)
		if err != nil {
			return nil, err
		}

		fields = append(fields,
			interpreter.NewCompositeField(
				inter,
				fieldType.Identifier,
				importedFieldValue,
			),
		)
	}

	if location == nil {
		switch sema.NativeCompositeTypes[qualifiedIdentifier] {
		case sema.PublicKeyType:
			// PublicKey has a dedicated constructor
			// (e.g. it has computed fields that must be initialized)
			return importPublicKey(inter, fields, getLocationRange)

		case sema.HashAlgorithmType:
			// HashAlgorithmType has a dedicated constructor
			// (e.g. it has host functions)
			return importHashAlgorithm(inter, fields)

		case sema.SignatureAlgorithmType:
			// SignatureAlgorithmType has a dedicated constructor
			// (e.g. it has host functions)
			return importSignatureAlgorithm(inter, fields)

		default:
			return nil, errors.NewDefaultUserError(
				"cannot import value of type %s",
				qualifiedIdentifier,
			)
		}
	}

	return interpreter.NewCompositeValue(
		inter,
		getLocationRange,
		location,
		qualifiedIdentifier,
		kind,
		fields,
		common.Address{},
	), nil
}

func importPublicKey(
	inter *interpreter.Interpreter,
	fields []interpreter.CompositeField,
	getLocationRange func() interpreter.LocationRange,
) (
	*interpreter.CompositeValue,
	error,
) {

	var publicKeyValue *interpreter.ArrayValue
	var signAlgoValue *interpreter.CompositeValue

	ty := sema.PublicKeyType

	for _, field := range fields {
		switch field.Name {
		case sema.PublicKeyPublicKeyField:
			arrayValue, ok := field.Value.(*interpreter.ArrayValue)
			if !ok {
				return nil, errors.NewDefaultUserError(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					field.Name,
					field.Value,
				)
			}

			publicKeyValue = arrayValue

		case sema.PublicKeySignAlgoField:
			compositeValue, ok := field.Value.(*interpreter.CompositeValue)
			if !ok {
				return nil, errors.NewDefaultUserError(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					field.Name,
					field.Value,
				)
			}

			signAlgoValue = compositeValue

		default:
			return nil, errors.NewDefaultUserError(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				field.Name,
			)
		}

	}

	if publicKeyValue == nil {
		return nil, errors.NewDefaultUserError(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.PublicKeyPublicKeyField,
		)
	}

	if signAlgoValue == nil {
		return nil, errors.NewDefaultUserError(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.PublicKeySignAlgoField,
		)
	}

	return interpreter.NewPublicKeyValue(
		inter,
		getLocationRange,
		publicKeyValue,
		signAlgoValue,
		inter.PublicKeyValidationHandler,
	), nil
}

func importHashAlgorithm(
	inter *interpreter.Interpreter,
	fields []interpreter.CompositeField,
) (
	*interpreter.CompositeValue,
	error,
) {

	var foundRawValue bool
	var rawValue interpreter.UInt8Value

	ty := sema.HashAlgorithmType

	for _, field := range fields {
		switch field.Name {
		case sema.EnumRawValueFieldName:
			rawValue, foundRawValue = field.Value.(interpreter.UInt8Value)
			if !foundRawValue {
				return nil, errors.NewDefaultUserError(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					field.Name,
					field.Value,
				)
			}

		default:
			return nil, errors.NewDefaultUserError(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				field.Name,
			)
		}
	}

	if !foundRawValue {
		return nil, errors.NewDefaultUserError(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.EnumRawValueFieldName,
		)
	}

	return stdlib.NewHashAlgorithmCase(inter, uint8(rawValue)), nil
}

func importSignatureAlgorithm(
	inter *interpreter.Interpreter,
	fields []interpreter.CompositeField,
) (
	*interpreter.CompositeValue,
	error,
) {

	var foundRawValue bool
	var rawValue interpreter.UInt8Value

	ty := sema.SignatureAlgorithmType

	for _, field := range fields {
		switch field.Name {
		case sema.EnumRawValueFieldName:
			rawValue, foundRawValue = field.Value.(interpreter.UInt8Value)
			if !foundRawValue {
				return nil, errors.NewDefaultUserError(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					field.Name,
					field.Value,
				)
			}

		default:
			return nil, errors.NewDefaultUserError(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				field.Name,
			)
		}
	}

	if !foundRawValue {
		return nil, errors.NewDefaultUserError(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.EnumRawValueFieldName,
		)
	}

	return stdlib.NewSignatureAlgorithmCase(inter, uint8(rawValue)), nil
}
