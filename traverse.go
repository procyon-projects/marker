package marker

import (
	"errors"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
)

type Kind int

const (
	AnyKind Kind = iota
	Object
	Array
	Chan
	Map
	Ptr
	Variadic
	Function
	Interface
	Struct
	UserDefined
)

type Type interface {
	Kind() Kind
	Compare(typ Type) bool
}

type ValueType struct {
	ImportName string
	Name       string
}

type ConstValue struct {
	Name         string
	Value        string
	IsExported   bool
	Type         *ValueType
	Position     Position
	RawFile      *ast.File
	RawValueSpec *ast.ValueSpec
}

type TypeInfo struct {
	Name     string
	Type     Type
	Markers  MarkerValues
	RawField *ast.Field
}

func (typeInfo TypeInfo) Compare(another TypeInfo) bool {
	return false
}

type FileCallback func(file *File, err error)

type Position struct {
	Line   int
	Column int
}

func (pos Position) Compare(another Position) bool {
	return pos.Line == another.Line && pos.Column == another.Line
}

type Import struct {
	Name          string
	Path          string
	Position      Position
	RawImportSpec *ast.ImportSpec
}

type File struct {
	Name          string
	FullPath      string
	Package       *Package
	Imports       []Import
	Consts        []ConstValue
	Markers       MarkerValues
	ImportMarkers []MarkerValues

	FunctionTypes    []FunctionType
	StructTypes      []StructType
	InterfaceTypes   []InterfaceType
	UserDefinedTypes []UserDefinedType
	RawFile          *ast.File
}

func (file *File) Compare(another *File) bool {
	if file.Package != another.Package && !file.Package.Compare(another.Package) {
		return false
	}

	if file.FullPath != another.FullPath || file.Name != another.Name {
		return false
	}

	return true
}

type AnyKindType struct {
}

func (typ AnyKindType) Kind() Kind {
	return AnyKind
}

func (typ AnyKindType) Compare(another Type) bool {
	if typ.Kind() != another.Kind() {
		return false
	}

	_, ok := another.(AnyKindType)

	if !ok {
		return false
	}

	return true
}

type ObjectType struct {
	ImportName string
	Name       string
}

func (typ ObjectType) Kind() Kind {
	return Object
}

func (typ ObjectType) Compare(another Type) bool {
	if typ.Kind() != another.Kind() {
		return false
	}

	return true
}

type PointerType struct {
	Typ Type
}

func (typ PointerType) Kind() Kind {
	return Ptr
}

func (typ PointerType) Compare(another Type) bool {
	if typ.Kind() != another.Kind() {
		return false
	}

	pointerType, ok := another.(PointerType)

	if !ok {
		return false
	}

	return typ.Typ.Compare(pointerType)
}

type ArrayType struct {
	ItemType Type
}

func (typ ArrayType) Kind() Kind {
	return Array
}

func (typ ArrayType) Compare(another Type) bool {
	if typ.Kind() != another.Kind() {
		return false
	}

	arrayType, ok := another.(ArrayType)

	if !ok {
		return false
	}

	return typ.ItemType.Compare(arrayType)
}

type DictionaryType struct {
	KeyType   Type
	ValueType Type
}

func (typ DictionaryType) Kind() Kind {
	return Map
}

func (typ DictionaryType) Compare(another Type) bool {
	if typ.Kind() != another.Kind() {
		return false
	}

	dictionaryType, ok := another.(DictionaryType)

	if !ok {
		return false
	}

	return typ.KeyType.Compare(dictionaryType.KeyType) && typ.ValueType.Compare(dictionaryType.ValueType)
}

type ChanDirection int

const (
	SEND ChanDirection = 1 << iota
	RECEIVE
)

type ChanType struct {
	Typ       Type
	Direction ChanDirection
}

func (typ ChanType) Kind() Kind {
	return Chan
}

func (typ ChanType) Compare(another Type) bool {
	if typ.Kind() != another.Kind() {
		return false
	}

	chanType, ok := another.(ChanType)

	if !ok {
		return false
	}

	if typ.Direction != chanType.Direction {
		return false
	}

	return typ.Typ.Compare(chanType.Typ)
}

type FunctionType struct {
	Name         string
	IsExported   bool
	Position     Position
	Markers      MarkerValues
	Parameters   []TypeInfo
	ReturnValues []TypeInfo
	File         *File
	RawFile      *ast.File
	RawFuncDecl  *ast.FuncDecl
	RawFuncType  *ast.FuncType
}

func (typ FunctionType) Kind() Kind {
	return Function
}

func (typ FunctionType) Compare(another Type) bool {
	if typ.Kind() != another.Kind() {
		return false
	}

	funcType, ok := another.(FunctionType)

	if !ok {
		return false
	}

	if typ.File == nil || funcType.File == nil || !typ.File.Compare(funcType.File) {
		return false
	}

	if typ.Name != funcType.Name || typ.IsExported != funcType.IsExported {
		return false
	}

	if !typ.Position.Compare(funcType.Position) {
		return false
	}

	if len(typ.Parameters) != len(funcType.Parameters) || len(typ.ReturnValues) != len(funcType.ReturnValues) {
		return false
	}

	for index, parameter := range typ.Parameters {
		if !parameter.Compare(funcType.Parameters[index]) {
			return false
		}
	}

	for index, returnValue := range typ.ReturnValues {
		if !returnValue.Compare(funcType.ReturnValues[index]) {
			return false
		}
	}

	return true
}

type Field struct {
	Name       string
	IsExported bool
	IsEmbedded bool
	Position   Position
	Markers    MarkerValues
	Type       Type
	File       *File
	RawFile    *ast.File
	RawField   *ast.Field
}

func (field Field) Compare(another Field) bool {
	if field.File == nil || another.File == nil || !field.File.Compare(another.File) {
		return false
	}

	if field.Name != another.Name || field.IsExported != another.IsExported || field.IsEmbedded != another.IsEmbedded {
		return false
	}

	if !field.Position.Compare(another.Position) {
		return false
	}

	if field.Type == nil || another.Type == nil || field.Type.Compare(another.Type) {
		return false
	}

	return true
}

type Method struct {
	Name         string
	IsExported   bool
	Position     Position
	Markers      MarkerValues
	Receiver     TypeInfo
	Parameters   []TypeInfo
	ReturnValues []TypeInfo
	File         *File
	RawFile      *ast.File
	RawField     *ast.Field
	RawFuncDecl  *ast.FuncDecl
	RawFuncType  *ast.FuncType
}

func (method Method) Compare(another Method) bool {
	if method.File == nil || another.File == nil || !method.File.Compare(another.File) {
		return false
	}

	if method.Name != another.Name || method.IsExported != another.IsExported {
		return false
	}

	if !method.Position.Compare(another.Position) {
		return false
	}

	if !method.Receiver.Compare(another.Receiver) {
		return false
	}

	if len(method.Parameters) != len(another.Parameters) || len(method.ReturnValues) != len(another.ReturnValues) {
		return false
	}

	for index, parameter := range method.Parameters {
		if !parameter.Compare(another.Parameters[index]) {
			return false
		}
	}

	for index, returnValue := range method.ReturnValues {
		if !returnValue.Compare(another.ReturnValues[index]) {
			return false
		}
	}

	return true
}

type StructType struct {
	Name        string
	IsExported  bool
	Position    Position
	Markers     MarkerValues
	Fields      []Field
	Methods     []Method
	File        *File
	RawFile     *ast.File
	RawGenDecl  *ast.GenDecl
	RawTypeSpec *ast.TypeSpec
}

func (typ StructType) Kind() Kind {
	return Struct
}

func (typ StructType) Compare(another Type) bool {
	if typ.Kind() != another.Kind() {
		return false
	}

	structType, ok := another.(StructType)

	if !ok {
		return false
	}

	if !typ.File.Compare(structType.File) {
		return false
	}

	if typ.Name != structType.Name || typ.IsExported != structType.IsExported || !typ.Position.Compare(structType.Position) {
		return false
	}

	if len(typ.Fields) != len(structType.Fields) || len(typ.Methods) != len(structType.Methods) {
		return false
	}

	for index, field := range typ.Fields {
		if !field.Compare(structType.Fields[index]) {
			return false
		}
	}

	for index, method := range typ.Methods {
		if !method.Compare(structType.Methods[index]) {
			return false
		}
	}

	return true
}

func (typ StructType) Implements(interfaceType InterfaceType) bool {

	return true
}

type UserDefinedType struct {
	Name        string
	IsExported  bool
	ActualType  Type
	Position    Position
	Markers     MarkerValues
	Methods     []Method
	File        *File
	RawFile     *ast.File
	RawGenDecl  *ast.GenDecl
	RawTypeSpec *ast.TypeSpec
}

func (typ UserDefinedType) Kind() Kind {
	return UserDefined
}

func (typ UserDefinedType) Compare(another Type) bool {
	if typ.Kind() != another.Kind() {
		return false
	}

	return true
}

type InterfaceType struct {
	Name        string
	IsExported  bool
	Position    Position
	Markers     MarkerValues
	Methods     []Method
	File        *File
	RawFile     *ast.File
	RawGenDecl  *ast.GenDecl
	RawTypeSpec *ast.TypeSpec
}

func (typ InterfaceType) Kind() Kind {
	return Interface
}

func (typ InterfaceType) Compare(another Type) bool {
	if typ.Kind() != another.Kind() {
		return false
	}

	interfaceType, ok := another.(InterfaceType)

	if !ok {
		return false
	}

	if !typ.File.Compare(interfaceType.File) {
		return false
	}

	if typ.Name != interfaceType.Name || typ.IsExported != interfaceType.IsExported || !typ.Position.Compare(interfaceType.Position) {
		return false
	}

	if len(typ.Methods) != len(interfaceType.Methods) {
		return false
	}

	for index, method := range typ.Methods {
		if !method.Compare(interfaceType.Methods[index]) {
			return false
		}
	}

	return true
}

type AnonymousStructType struct {
	Fields []Field
}

func (typ AnonymousStructType) Kind() Kind {
	return Struct
}

func (typ AnonymousStructType) Compare(another Type) bool {
	if typ.Kind() != another.Kind() {
		return false
	}

	return true
}

type VariadicType struct {
	ItemType Type
}

func (typ VariadicType) Kind() Kind {
	return Variadic
}

func (typ VariadicType) Compare(another Type) bool {
	if typ.Kind() != another.Kind() {
		return false
	}

	return false
}

func EachFile(collector *Collector, pkgs []*Package, callback FileCallback) {
	if collector == nil {
		callback(nil, errors.New("collector cannot be nil"))
	}

	if pkgs == nil {
		callback(nil, errors.New("pkgs(packages) cannot be nil"))
	}

	var fileMap = make(map[*ast.File]*File)
	var errs []error

	for _, pkg := range pkgs {
		markers, err := collector.Collect(pkg)

		if err != nil {
			errs = append(errs, err.(ErrorList)...)
			continue
		}

		fileNodeMap := eachPackage(pkg, markers)

		for fileNode, file := range fileNodeMap {
			fileMap[fileNode] = file
		}
	}

	if errs != nil {
		callback(nil, NewErrorList(errs))
	}

	for _, file := range fileMap {
		callback(file, nil)
	}
}

func eachPackage(pkg *Package, markers map[ast.Node]MarkerValues) map[*ast.File]*File {
	var fileNodeMap = make(map[*ast.File]*File)
	var methods = make([]Method, 0)

	visitPackageFiles(pkg, func(file *ast.File) {
		_, ok := fileNodeMap[file]

		if ok {
			return
		}

		fileNodeMap[file] = getFile(pkg, file, markers)
	}, func(file *ast.File, decl *ast.GenDecl) {
		fileInfo, ok := fileNodeMap[file]

		if !ok {
			return
		}

		if markerValues, ok := markers[decl]; ok {
			fileInfo.ImportMarkers = append(fileInfo.ImportMarkers, markerValues)
		}
	}, func(file *ast.File, decl *ast.GenDecl) {
		fileInfo, ok := fileNodeMap[file]

		if !ok {
			return
		}

		constValues := getConstValues(pkg.Fset, file, decl.Specs)

		if constValues != nil {
			fileInfo.Consts = append(fileInfo.Consts, constValues...)
		}
	}, func(file *ast.File, decl *ast.GenDecl, spec *ast.TypeSpec) {
		fileInfo, ok := fileNodeMap[file]

		if !ok {
			return
		}

		typ := getType(pkg.Fset, fileInfo, file, decl, spec, markers)

		switch spec.Type.(type) {
		case *ast.InterfaceType:
			fileInfo.InterfaceTypes = append(fileInfo.InterfaceTypes, typ.(InterfaceType))
		case *ast.StructType:
			fileInfo.StructTypes = append(fileInfo.StructTypes, typ.(StructType))
		default:
			fileInfo.UserDefinedTypes = append(fileInfo.UserDefinedTypes, typ.(UserDefinedType))
		}
	}, func(file *ast.File, decl *ast.FuncDecl, funcType *ast.FuncType) {
		fileInfo, ok := fileNodeMap[file]

		if !ok {
			return
		}

		// If Recv is nil, it is a function, not a method
		if decl.Recv == nil {
			functionType := getFunctionType(pkg.Fset, fileInfo, file, decl, funcType, markers)
			fileInfo.FunctionTypes = append(fileInfo.FunctionTypes, functionType)
		} else {
			method := getMethod(pkg.Fset, fileInfo, file, decl, funcType, markers)
			methods = append(methods, method)
		}
	})

	resolveMethods(fileNodeMap, methods)

	return fileNodeMap
}

func resolveMethods(fileInfoMap map[*ast.File]*File, methods []Method) {

	for _, method := range methods {
		resolveMethod(fileInfoMap, method)
	}

}

func resolveMethod(fileInfoMap map[*ast.File]*File, method Method) {

	receiverType := method.Receiver.Type
	var receiverTypeName string

	switch typed := receiverType.(type) {
	case *PointerType:
		objectType := typed.Typ.(*ObjectType)

		if objectType.ImportName != "" {
			return
		}

		receiverTypeName = objectType.Name

	case *ObjectType:
		if typed.ImportName != "" {
			return
		}

		receiverTypeName = typed.Name
	}

	for file, fileInfo := range fileInfoMap {

		for index, structType := range fileInfo.StructTypes {

			if file.Name.Name == fileInfo.Package.Name && structType.Name == receiverTypeName {
				fileInfo.StructTypes[index].Methods = append(fileInfo.StructTypes[index].Methods, method)
				return
			}

		}

		for index, userDefinedType := range fileInfo.UserDefinedTypes {

			if file.Name.Name == fileInfo.Package.Name && userDefinedType.Name == receiverTypeName {
				fileInfo.UserDefinedTypes[index].Methods = append(fileInfo.UserDefinedTypes[index].Methods, method)
				return
			}

		}
	}
}

func getFile(pkg *Package, file *ast.File, markers map[ast.Node]MarkerValues) *File {
	position := pkg.Fset.Position(file.Pos())
	fileFullPath := position.Filename

	return &File{
		Name:           filepath.Base(fileFullPath),
		FullPath:       fileFullPath,
		Package:        pkg,
		Imports:        getFileImports(pkg.Fset, file),
		Consts:         make([]ConstValue, 0),
		Markers:        markers[file],
		ImportMarkers:  make([]MarkerValues, 0),
		FunctionTypes:  make([]FunctionType, 0),
		StructTypes:    make([]StructType, 0),
		InterfaceTypes: make([]InterfaceType, 0),
		RawFile:        file,
	}
}

func getFileImports(fileSet *token.FileSet, file *ast.File) []Import {
	imports := make([]Import, 0)

	for _, importInfo := range file.Imports {
		importPosition := fileSet.Position(importInfo.Pos())
		importName := ""

		if importInfo.Name != nil {
			importName = importInfo.Name.Name
		}

		imports = append(imports, Import{
			Name: importName,
			Path: importInfo.Path.Value[1 : len(importInfo.Path.Value)-1],
			Position: Position{
				importPosition.Line,
				importPosition.Column,
			},
			RawImportSpec: importInfo,
		})
	}

	return imports
}

func getFunctionType(fileSet *token.FileSet,
	fileInfo *File,
	file *ast.File,
	decl *ast.FuncDecl,
	funcType *ast.FuncType,
	markers map[ast.Node]MarkerValues) FunctionType {

	function := &FunctionType{
		Position:    getPosition(fileSet, funcType.Pos()),
		File:        fileInfo,
		RawFile:     file,
		RawFuncDecl: decl,
		RawFuncType: funcType,
	}

	if decl != nil {
		function.Name = decl.Name.Name
		function.IsExported = ast.IsExported(decl.Name.Name)
		function.Markers = markers[decl]
	}

	if funcType.Params != nil {
		function.Parameters = getFieldTypesInfo(fileSet, fileInfo, file, funcType.Params.List, markers)
	}

	if funcType.Results != nil {
		function.ReturnValues = getFieldTypesInfo(fileSet, fileInfo, file, funcType.Results.List, markers)
	}

	return *function
}

func getConstValues(fileSet *token.FileSet,
	file *ast.File,
	specs []ast.Spec) []ConstValue {
	if specs == nil {
		return nil
	}

	constValues := make([]ConstValue, 0)

	var previousValueType *ValueType
	for _, spec := range specs {
		valueSpec := spec.(*ast.ValueSpec)
		constValue, inferredTypeName := getConstValue(fileSet, file, valueSpec)

		if valueSpec.Type == nil && constValue.Type == nil && inferredTypeName != "" {
			constValue.Type = &ValueType{
				Name: inferredTypeName,
			}
		} else if constValue.Type != nil {
			previousValueType = constValue.Type
		} else {
			constValue.Type = previousValueType
		}

		constValues = append(constValues, *constValue)

	}

	return constValues
}

func getConstValue(fileSet *token.FileSet, file *ast.File, spec *ast.ValueSpec) (*ConstValue, string) {
	constValue := &ConstValue{
		Name:         spec.Names[0].Name,
		Type:         getConstValueType(spec.Type),
		IsExported:   ast.IsExported(spec.Names[0].Name),
		Position:     getPosition(fileSet, spec.Pos()),
		RawFile:      file,
		RawValueSpec: spec,
	}

	inferredTypeName := ""

	if spec.Values != nil && len(spec.Values) > 0 {
		switch typedValue := spec.Values[0].(type) {
		case *ast.Ident:
			value := typedValue.Name

			if !strings.HasPrefix(value, "\"") && !strings.HasSuffix(value, "\"") {
				if value == "true" || value == "false" {
					inferredTypeName = "bool"
					constValue.Value = value
				}
			}

		case *ast.BasicLit:
			constValue.Value = typedValue.Value

			switch typedValue.Kind {
			case token.STRING:
				inferredTypeName = "string"
			case token.INT:
				inferredTypeName = "int"
			case token.FLOAT:
				inferredTypeName = "float"
			case token.CHAR:
				inferredTypeName = "char"
			}
		}
	}

	return constValue, inferredTypeName
}

func getConstValueType(typ ast.Expr) *ValueType {
	switch typedExpr := typ.(type) {
	case *ast.Ident:
		return &ValueType{
			Name: typedExpr.Name,
		}
	case *ast.SelectorExpr:
		return &ValueType{
			ImportName: typedExpr.X.(*ast.Ident).Name,
			Name:       typedExpr.Sel.Name,
		}
	}

	return nil
}

func getType(fileSet *token.FileSet,
	fileInfo *File,
	file *ast.File,
	decl *ast.GenDecl,
	spec *ast.TypeSpec,
	markers map[ast.Node]MarkerValues) Type {

	var typ Type

	switch specType := spec.Type.(type) {
	case *ast.InterfaceType:
		interfaceType := InterfaceType{
			Name:        spec.Name.Name,
			IsExported:  ast.IsExported(spec.Name.Name),
			Position:    getPosition(fileSet, spec.Pos()),
			Markers:     markers[spec],
			File:        fileInfo,
			RawFile:     file,
			RawGenDecl:  decl,
			RawTypeSpec: spec,
		}

		interfaceType.Methods = getInterfaceMethods(fileSet, fileInfo, file, specType, markers)
		typ = interfaceType
	case *ast.StructType:
		structType := StructType{
			Name:        spec.Name.Name,
			IsExported:  ast.IsExported(spec.Name.Name),
			Position:    getPosition(fileSet, spec.Pos()),
			Markers:     markers[spec],
			File:        fileInfo,
			RawFile:     file,
			RawGenDecl:  decl,
			RawTypeSpec: spec,
		}

		structType.Fields = getStructFields(fileSet, fileInfo, file, specType, markers)
		typ = structType
	case *ast.Ident, *ast.StarExpr, *ast.SelectorExpr, *ast.MapType, *ast.ArrayType, *ast.ChanType, *ast.FuncType:
		typ = UserDefinedType{
			Name:        spec.Name.Name,
			IsExported:  ast.IsExported(spec.Name.Name),
			Position:    getPosition(fileSet, spec.Pos()),
			ActualType:  getTypeFromExpression(fileSet, fileInfo, file, spec.Type, markers),
			Markers:     markers[spec],
			File:        fileInfo,
			RawFile:     file,
			RawGenDecl:  decl,
			RawTypeSpec: spec,
		}
	}

	return typ
}

func getInterfaceMethods(fileSet *token.FileSet,
	fileInfo *File,
	file *ast.File,
	specType *ast.InterfaceType,
	markers map[ast.Node]MarkerValues) []Method {

	methods := make([]Method, 0)

	for _, methodInfo := range specType.Methods.List {

		method := &Method{
			Name:        methodInfo.Names[0].Name,
			IsExported:  ast.IsExported(methodInfo.Names[0].Name),
			Position:    getPosition(fileSet, methodInfo.Pos()),
			Markers:     markers[methodInfo],
			File:        fileInfo,
			RawFile:     file,
			RawFuncType: methodInfo.Type.(*ast.FuncType),
			RawField:    methodInfo,
		}

		if methodInfo.Type.(*ast.FuncType).Params != nil {
			method.Parameters = getFieldTypesInfo(fileSet, fileInfo, file, methodInfo.Type.(*ast.FuncType).Params.List, markers)
		}

		if methodInfo.Type.(*ast.FuncType).Results != nil {
			method.ReturnValues = getFieldTypesInfo(fileSet, fileInfo, file, methodInfo.Type.(*ast.FuncType).Results.List, markers)
		}

		methods = append(methods, *method)

	}

	return methods
}

func getStructFields(fileSet *token.FileSet,
	fileInfo *File,
	file *ast.File,
	specType *ast.StructType,
	markers map[ast.Node]MarkerValues) []Field {

	fields := make([]Field, 0)

	for _, fieldTypeInfo := range specType.Fields.List {

		if fieldTypeInfo.Names == nil {
			field := &Field{
				IsEmbedded: true,
				Position:   getPosition(fileSet, fieldTypeInfo.Type.Pos()),
				Markers:    markers[fieldTypeInfo],
				Type:       getTypeFromExpression(fileSet, fileInfo, file, fieldTypeInfo.Type, markers),
				File:       fileInfo,
				RawFile:    file,
				RawField:   fieldTypeInfo,
			}
			fields = append(fields, *field)
			continue
		}

		for _, fieldName := range fieldTypeInfo.Names {
			field := &Field{
				Name:       fieldName.Name,
				IsExported: ast.IsExported(fieldName.Name),
				IsEmbedded: false,
				Position:   getPosition(fileSet, fieldName.Pos()),
				Markers:    markers[fieldTypeInfo],
				Type:       getTypeFromExpression(fileSet, fileInfo, file, fieldTypeInfo.Type, markers),
				File:       fileInfo,
				RawFile:    file,
				RawField:   fieldTypeInfo,
			}

			fields = append(fields, *field)
		}

	}

	return fields
}

func getMethod(fileSet *token.FileSet,
	fileInfo *File,
	file *ast.File,
	decl *ast.FuncDecl,
	funcType *ast.FuncType,
	markers map[ast.Node]MarkerValues) Method {

	method := &Method{
		Name:        decl.Name.Name,
		IsExported:  ast.IsExported(decl.Name.Name),
		Position:    getPosition(fileSet, funcType.Pos()),
		Markers:     markers[decl],
		File:        fileInfo,
		RawFile:     file,
		RawFuncDecl: decl,
		RawFuncType: funcType,
	}

	if funcType.Params != nil {
		method.Parameters = getFieldTypesInfo(fileSet, fileInfo, file, funcType.Params.List, markers)
	}

	if funcType.Results != nil {
		method.ReturnValues = getFieldTypesInfo(fileSet, fileInfo, file, funcType.Results.List, markers)
	}

	// Receiver
	receiver := decl.Recv.List[0]
	receiverType := getTypeFromExpression(fileSet, fileInfo, file, receiver.Type, markers)

	method.Receiver = TypeInfo{
		Name:     receiver.Names[0].Name,
		Type:     receiverType,
		Markers:  nil,
		RawField: nil,
	}

	return *method
}

func getPosition(tokenFileSet *token.FileSet, pos token.Pos) Position {
	position := tokenFileSet.Position(pos)
	return Position{
		Line:   position.Line,
		Column: position.Column,
	}
}

func getFieldTypesInfo(tokenFileSet *token.FileSet,
	fileInfo *File,
	file *ast.File,
	fieldList []*ast.Field, markers map[ast.Node]MarkerValues) []TypeInfo {
	types := make([]TypeInfo, 0)

	for _, field := range fieldList {

		if field.Names == nil {
			typeInfo := &TypeInfo{
				RawField: field,
				Type:     getTypeFromExpression(tokenFileSet, fileInfo, file, field.Type, markers),
				Markers:  markers[field],
			}

			types = append(types, *typeInfo)
			continue
		}

		for _, name := range field.Names {
			typeInfo := &TypeInfo{
				Name:     name.Name,
				RawField: field,
				Type:     getTypeFromExpression(tokenFileSet, fileInfo, file, field.Type, markers),
				Markers:  markers[field],
			}

			types = append(types, *typeInfo)
		}
	}

	return types
}

func getTypeFromExpression(tokenFileSet *token.FileSet,
	fileInfo *File,
	file *ast.File,
	expression ast.Expr,
	markers map[ast.Node]MarkerValues) Type {

	switch result := expression.(type) {
	case *ast.Ident:
		return &ObjectType{
			Name: result.Name,
		}
	case *ast.SelectorExpr:
		return &ObjectType{
			ImportName: result.X.(*ast.Ident).Name,
			Name:       result.Sel.Name,
		}
	case *ast.StarExpr:
		return &PointerType{
			Typ: getTypeFromExpression(tokenFileSet, fileInfo, file, result.X, markers),
		}
	case *ast.ArrayType:
		return &ArrayType{
			ItemType: getTypeFromExpression(tokenFileSet, fileInfo, file, result.Elt, markers),
		}
	case *ast.MapType:
		return &DictionaryType{
			KeyType:   getTypeFromExpression(tokenFileSet, fileInfo, file, result.Key, markers),
			ValueType: getTypeFromExpression(tokenFileSet, fileInfo, file, result.Value, markers),
		}
	case *ast.ChanType:
		chanTyp := &ChanType{
			Typ: getTypeFromExpression(tokenFileSet, fileInfo, file, result.Value, markers),
		}

		if result.Dir&ast.SEND == ast.SEND {
			chanTyp.Direction |= SEND
		}

		if result.Dir&ast.RECV == ast.RECV {
			chanTyp.Direction |= RECEIVE
		}

		return chanTyp
	case *ast.StructType:
		anonymousStructType := &AnonymousStructType{}

		fieldTypeInfoList := getFieldTypesInfo(tokenFileSet, fileInfo, file, result.Fields.List, markers)

		for _, fieldTypeInfo := range fieldTypeInfoList {

			field := &Field{
				Name:       fieldTypeInfo.Name,
				IsExported: ast.IsExported(fieldTypeInfo.Name),
				Position:   getPosition(tokenFileSet, fieldTypeInfo.RawField.Pos()),
				Markers:    fieldTypeInfo.Markers,
				Type:       fieldTypeInfo.Type,
				File:       fileInfo,
				RawFile:    file,
				RawField:   fieldTypeInfo.RawField,
			}

			anonymousStructType.Fields = append(anonymousStructType.Fields, *field)

		}

		return anonymousStructType
	case *ast.InterfaceType:
		return &AnyKindType{}
	case *ast.FuncType:
		return getFunctionType(tokenFileSet, fileInfo, file, nil, result, markers)
	case *ast.Ellipsis:
		return &VariadicType{
			ItemType: getTypeFromExpression(tokenFileSet, fileInfo, file, result.Elt, markers),
		}
	}

	panic("Unreachable code!")
}
