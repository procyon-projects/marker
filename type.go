package marker

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

type ArgumentType int

func (argumentType ArgumentType) String() string {
	return argumentTypeText[argumentType]
}

const (
	InvalidType ArgumentType = iota
	RawType
	AnyType
	BoolType
	IntegerType
	StringType
	SliceType
	MapType
)

var argumentTypeText = map[ArgumentType]string{
	InvalidType: "InvalidType",
	RawType:     "RawType",
	AnyType:     "AnyType",
	BoolType:    "BoolType",
	IntegerType: "IntegerType",
	StringType:  "StringType",
	SliceType:   "SliceType",
	MapType:     "MapType",
}

var (
	interfaceType = reflect.TypeOf((*interface{})(nil)).Elem()
	rawType       = reflect.TypeOf((*[]byte)(nil)).Elem()
)

type ArgumentTypeInfo struct {
	ActualType ArgumentType
	ItemType   *ArgumentTypeInfo
}

func GetArgumentTypeInfo(typ reflect.Type) (ArgumentTypeInfo, error) {
	typeInfo := &ArgumentTypeInfo{}

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ == rawType {
		typeInfo.ActualType = RawType
		return *typeInfo, nil
	}

	if typ == interfaceType {
		typeInfo.ActualType = AnyType
		return *typeInfo, nil
	}

	switch typ.Kind() {
	case reflect.String:
		typeInfo.ActualType = StringType
	case reflect.Uint8, reflect.Uint16, reflect.Uint, reflect.Uint32, reflect.Uint64:
		typeInfo.ActualType = IntegerType
	case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32, reflect.Int64:
		typeInfo.ActualType = IntegerType
	case reflect.Bool:
		typeInfo.ActualType = BoolType
	case reflect.Slice:
		typeInfo.ActualType = SliceType
		itemType, err := GetArgumentTypeInfo(typ.Elem())

		if err != nil {
			return ArgumentTypeInfo{}, fmt.Errorf("bad slice item type: %w", err)
		}

		typeInfo.ItemType = &itemType
	case reflect.Map:
		if typ.Key().Kind() != reflect.String {
			return ArgumentTypeInfo{}, fmt.Errorf("map key must be string")
		}

		typeInfo.ActualType = MapType
		itemType, err := GetArgumentTypeInfo(typ.Elem())

		if err != nil {
			return ArgumentTypeInfo{}, fmt.Errorf("bad map item type: %w", err)
		}

		typeInfo.ItemType = &itemType
	default:
		return ArgumentTypeInfo{}, fmt.Errorf("type has unsupported kind %s", typ.Kind())
	}

	return *typeInfo, nil
}

func (typeInfo ArgumentTypeInfo) Parse(scanner *Scanner, out reflect.Value) error {
	switch typeInfo.ActualType {
	case BoolType:
		return typeInfo.parseBoolean(scanner, out)
	case IntegerType:
		return typeInfo.parseInteger(scanner, out)
	case StringType:
		return typeInfo.parseString(scanner, out)
	case SliceType:
		return typeInfo.parseSlice(scanner, out)
	case MapType:
		return typeInfo.parseMap(scanner, out)
	case AnyType:
		inferredType, _ := typeInfo.inferType(scanner, out, false)
		newOut := out

		switch inferredType.ActualType {
		case SliceType:
			newType, err := inferredType.makeSliceType()

			if err != nil {
				return err
			}

			newOut = reflect.Indirect(reflect.New(newType))
		case MapType:
			newType, err := inferredType.makeMapType()

			if err != nil {
				return err
			}

			newOut = reflect.Indirect(reflect.New(newType))
		}

		if newOut.Kind() == reflect.Ptr {
			newOut = newOut.Elem()
		}

		if !newOut.CanSet() {
			return nil
		}

		err := inferredType.Parse(scanner, newOut)

		if err != nil {
			return err
		}

		inferredType.setValue(out, newOut)
	}

	return nil
}

func (typeInfo ArgumentTypeInfo) setValue(out, value reflect.Value) {
	outType := out.Type()

	if outType.Kind() == reflect.Ptr {
		outType = outType.Elem()
		out = out.Elem()
	}

	if outType != value.Type() {
		value = value.Convert(outType)
	}

	out.Set(value)
}

func (typeInfo ArgumentTypeInfo) parseBoolean(scanner *Scanner, out reflect.Value) error {
	if scanner == nil {
		return errors.New("scanner cannot be nil")
	}

	if !scanner.Expect(Identifier, "Boolean (true or false)") {
		return nil
	}

	switch scanner.Token() {
	case "false":
		typeInfo.setValue(out, reflect.ValueOf(false))
	case "true":
		typeInfo.setValue(out, reflect.ValueOf(true))
	}

	return fmt.Errorf("expected true or false, got %q", scanner.Token())
}

func (typeInfo ArgumentTypeInfo) parseInteger(scanner *Scanner, out reflect.Value) error {
	if scanner == nil {
		return errors.New("scanner cannot be nil")
	}

	nextCharacter := scanner.Peek()

	isNegative := false

	if nextCharacter == '-' {
		isNegative = true
		scanner.Scan()
	}

	if !scanner.Expect(Integer, "Integer") {
		return nil
	}

	text := scanner.Token()

	if isNegative {
		text = "-" + text
	}

	intValue, err := strconv.Atoi(text)

	typeInfo.setValue(out, reflect.ValueOf(intValue))

	if err != nil {
		return fmt.Errorf("unable to parse integer: %v", err)
	}

	return nil
}

func (typeInfo ArgumentTypeInfo) parseString(scanner *Scanner, out reflect.Value) error {
	if scanner == nil {
		return errors.New("scanner cannot be nil")
	}

	startPosition := scanner.searchIndex

	token := scanner.Scan()

	if token == String {

		value, err := strconv.Unquote(scanner.Token())

		if err != nil {
			return err
		}

		typeInfo.setValue(out, reflect.ValueOf(value))
		return nil
	}

	for character := scanner.SkipWhitespaces(); character != ',' && character != ';' && character != ':' && character != '}' && character != EOF; character = scanner.SkipWhitespaces() {
		scanner.Scan()
	}

	endPosition := scanner.searchIndex

	value := string(scanner.source[startPosition:endPosition])
	typeInfo.setValue(out, reflect.ValueOf(value))

	return nil
}

func (typeInfo ArgumentTypeInfo) parseSlice(scanner *Scanner, out reflect.Value) error {
	if scanner == nil {
		return errors.New("scanner cannot be nil")
	}

	sliceType := reflect.Zero(out.Type())
	sliceItemType := reflect.Indirect(reflect.New(out.Type().Elem()))

	if scanner.SkipWhitespaces() == '{' {

		scanner.Scan()

		for character := scanner.SkipWhitespaces(); character != '}' && character != EOF; character = scanner.SkipWhitespaces() {
			err := typeInfo.ItemType.Parse(scanner, sliceItemType)

			if err != nil {
				return err
			}

			sliceType = reflect.Append(sliceType, sliceItemType)

			token := scanner.SkipWhitespaces()

			if token == '}' {
				break
			}

			if !scanner.Expect(',', "Comma ','") {
				return nil
			}
		}

		if !scanner.Expect('}', "Right Curly Bracket '}'") {
			return nil
		}

		typeInfo.setValue(out, sliceType)
		return nil
	}

	for character := scanner.SkipWhitespaces(); character != ',' && character != '}' && character != EOF; character = scanner.SkipWhitespaces() {
		err := typeInfo.ItemType.Parse(scanner, sliceItemType)

		if err != nil {
			return err
		}

		sliceType = reflect.Append(sliceType, sliceItemType)

		token := scanner.SkipWhitespaces()

		if token == ',' || token == '}' || token == EOF {
			break
		}

		scanner.Scan()

		if token != ';' {
			return nil
		}
	}

	typeInfo.setValue(out, sliceType)
	return nil
}

func (typeInfo ArgumentTypeInfo) parseMap(scanner *Scanner, out reflect.Value) error {
	if scanner == nil {
		return errors.New("scanner cannot be nil")
	}

	mapType := reflect.MakeMap(out.Type())
	key := reflect.Indirect(reflect.New(out.Type().Key()))
	value := reflect.Indirect(reflect.New(out.Type().Elem()))

	if !scanner.Expect('{', "Left Curly Bracket") {
		return nil
	}

	for character := scanner.SkipWhitespaces(); character != '}' && character != EOF; character = scanner.SkipWhitespaces() {
		err := typeInfo.parseString(scanner, key)

		if err != nil {
			return err
		}

		if !scanner.Expect(':', "Colon ':'") {
			return nil
		}

		err = typeInfo.ItemType.Parse(scanner, value)

		if err != nil {
			return err
		}

		mapType.SetMapIndex(key, value)

		if scanner.SkipWhitespaces() == '}' {
			break
		}

		if !scanner.Expect(',', "Comma ','") {
			return nil
		}
	}

	if !scanner.Expect('}', "Right Curly Bracket '}'") {
		return nil
	}

	typeInfo.setValue(out, mapType)

	return nil
}

func (typeInfo ArgumentTypeInfo) inferType(scanner *Scanner, out reflect.Value, ignoreLegacySlice bool) (ArgumentTypeInfo, error) {

	character := scanner.SkipWhitespaces()
	searchIndex := scanner.searchIndex

	if !ignoreLegacySlice {
		itemType, _ := typeInfo.inferType(scanner, out, true)

		var token rune
		for token = scanner.Scan(); token != ',' && token != EOF && token != ';'; token = scanner.Scan() {
		}

		scanner.SetSearchIndex(searchIndex)

		if token == ';' {
			return ArgumentTypeInfo{
				ActualType: SliceType,
				ItemType:   &itemType,
			}, nil
		}

		return itemType, nil
	}

	switch character {
	case '"', '\'', '`':
		return ArgumentTypeInfo{
			ActualType: StringType,
		}, nil
	}

	if character == '{' {
		scanner.Scan()

		elementType, _ := typeInfo.inferType(scanner, out, true)

		// skip left curly bracket character
		scanner.SetSearchIndex(searchIndex + 1)

		if elementType.ActualType == StringType {

			var keyString string
			(ArgumentTypeInfo{ActualType: StringType}).parseString(scanner, reflect.Indirect(reflect.ValueOf(&keyString)))

			if scanner.Scan() == ':' {
				scanner.SetSearchIndex(searchIndex)

				return ArgumentTypeInfo{
					ActualType: MapType,
					ItemType: &ArgumentTypeInfo{
						ActualType: AnyType,
					},
				}, nil
			}
		}

		scanner.SetSearchIndex(searchIndex)

		return ArgumentTypeInfo{
			ActualType: SliceType,
			ItemType:   &elementType,
		}, nil
	}

	canBeString := false

	if character == 't' || character == 'f' {

		if token := scanner.Scan(); token == Identifier {

			switch scanner.Token() {
			case "true", "false":
				scanner.SetSearchIndex(searchIndex)
				return ArgumentTypeInfo{
					ActualType: BoolType,
				}, nil
			}

			canBeString = true
		} else {
			return ArgumentTypeInfo{
				ActualType: InvalidType,
			}, nil
		}
	}

	if !canBeString {

		token := scanner.Scan()

		if token == '-' {
			token = scanner.Scan()
		}

		if token == Integer {
			return ArgumentTypeInfo{
				ActualType: IntegerType,
			}, nil
		}

	}

	return ArgumentTypeInfo{
		ActualType: StringType,
	}, nil
}

func (typeInfo ArgumentTypeInfo) makeSliceType() (reflect.Type, error) {
	if typeInfo.ActualType != SliceType {
		return nil, errors.New("this is not slice type")
	}

	if typeInfo.ItemType == nil {
		return nil, errors.New("item type cannot be nil for slice type")
	}

	var itemType reflect.Type
	switch typeInfo.ItemType.ActualType {
	case IntegerType:
		itemType = reflect.TypeOf(int(0))
	case BoolType:
		itemType = reflect.TypeOf(false)
	case StringType:
		itemType = reflect.TypeOf("")
	case SliceType:
		subItemType, err := typeInfo.ItemType.makeSliceType()

		if err != nil {
			return nil, err
		}

		itemType = subItemType
	case MapType:
		subItemType, err := typeInfo.ItemType.makeMapType()

		if err != nil {
			return nil, err
		}

		itemType = subItemType
	default:
		return nil, fmt.Errorf("invalid type: %v", typeInfo.ItemType.ActualType)
	}

	return reflect.SliceOf(itemType), nil
}

func (typeInfo ArgumentTypeInfo) makeMapType() (reflect.Type, error) {
	if typeInfo.ActualType != MapType {
		return nil, errors.New("this is not map type")
	}

	if typeInfo.ItemType == nil {
		return nil, errors.New("item type cannot be nil for map type")
	}

	var itemType reflect.Type
	switch typeInfo.ItemType.ActualType {
	case IntegerType:
		itemType = reflect.TypeOf(int(0))
	case BoolType:
		itemType = reflect.TypeOf(false)
	case StringType:
		itemType = reflect.TypeOf("")
	case SliceType:
		subItemType, err := typeInfo.ItemType.makeSliceType()

		if err != nil {
			return nil, err
		}

		itemType = subItemType
	case MapType:
		subItemType, err := typeInfo.ItemType.makeMapType()
		if err != nil {
			return nil, err
		}
		itemType = subItemType
	case AnyType:
		itemType = interfaceType
	default:
		return nil, fmt.Errorf("invalid type: %v", typeInfo.ItemType.ActualType)
	}

	return reflect.MapOf(reflect.TypeOf(""), itemType), nil
}
