package main

import (
	"flag"
	"strings"
	"log"
	"go/parser"
	"go/token"
	"go/ast"
	"bytes"
	"path/filepath"
	"io/ioutil"
	"go/format"
)

const declarationTag = "Declaration"

type Event string
type State string

type StateDefinition struct {
	Name         State
	Events       map[Event]State
	Destinations map[State][]Event
	IsTerminal   bool
	Field        *ast.Field
}

type MachineDefinition struct {
	PkgName     string
	MachineName string
	States      map[State]StateDefinition
	Struct      *ast.StructType
}

func main() {
	typeNames := flag.String("type", "", "comma-separated list of type names; must be set")
	var dirName string
	flag.StringVar(&dirName, "dir", ".", "working directory; must be set")

	flag.Parse()
	if len(*typeNames) == 0 {
		log.Fatalf("the flag -type must be set")
	}
	if len(dirName) == 0 {
		log.Fatalf("the flag -dir must be set")
	}
	types := strings.Split(*typeNames, ",")
	verifySpecifiedTypes(types)

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dirName, nil, parser.SpuriousErrors)
	if err != nil {
		log.Fatal("can't parse destination dir ", err)
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, t := range types {
				obj := file.Scope.Lookup(t)
				if obj == nil {
					continue
				}
				generateStm(strings.TrimSuffix(t, declarationTag), dirName, pkg.Name, fset, obj)
			}
		}
	}
}

func generateStm(machineName string, dirName string, pkgName string, fset *token.FileSet, obj *ast.Object) {
	structType := extractStructTypeFromDefinition(fset, obj)
	states := map[State]StateDefinition{}
	for _, field := range structType.Fields.List {
		verifyField(fset, field)
		st := StateDefinition{
			Name:       State(field.Names[0].Name),
			IsTerminal: field.Tag == nil || field.Tag.Value == "",
			Field:      field,
		}
		st.Events, st.Destinations = parseStateMachineEventsAndDestinations(st, fset, field.Tag)
		states[st.Name] = st
	}
	definition := MachineDefinition{
		PkgName:     pkgName,
		MachineName: machineName,
		States:      states,
		Struct:      structType,
	}
	verifyDefinition(fset, definition)
	generateFromTemplateAndWriteToFile(definition, machineName, dirName)
}

func generateFromTemplateAndWriteToFile(definition MachineDefinition, machineName string, dirName string) {
	var b bytes.Buffer
	err := embeddedTemplate.Execute(&b, definition)
	if err != nil {
		log.Fatal("can't execute template ", err)
	}
	src, err := format.Source(b.Bytes())
	if err != nil {
		log.Fatal("can't format generated template ", err)
	}
	output := strings.ToLower(machineName + ".fsm.go")
	absPath, err := filepath.Abs("")
	if err != nil {
		log.Fatal("can't calculate abs path for: "+dirName, err)
	}
	outputPath := filepath.Join(absPath, output)
	log.Print(outputPath)
	err = ioutil.WriteFile(outputPath, src, 0664)
	if err != nil {
		log.Fatal("can't write file to disk. ", err)
	}
}

func parseStateMachineEventsAndDestinations(st StateDefinition, fset *token.FileSet, tag *ast.BasicLit) (map[Event]State, map[State][]Event) {
	if st.IsTerminal {
		return nil, nil
	}
	events := map[Event]State{}
	destinations := map[State][]Event{}
	eventsDeclarations := strings.Split(strip(tag.Value), ",")
	for _, eventDeclaration := range eventsDeclarations {
		eventStr := strings.Split(eventDeclaration, ":")
		if len(eventStr) != 2 || len(eventStr[0]) < 1 || len(eventStr[0]) < 3 {
			log.Fatalf("unsuported tag format %+v. %v", eventDeclaration, fset.Position(tag.Pos()))
		}
		ev := Event(eventStr[0])
		dst := State(strip(eventStr[1]))

		if ev == "Noop" {
			log.Fatalf("event `Noop` is reserved by system %+v", fset.Position(tag.Pos()))
		}

		if _, ok := events[ev]; ok {
			log.Fatalf("event `%s` duplicate on State `%s`. %v", ev, st.Name, fset.Position(tag.Pos()))
		}
		events[ev] = dst
		destinations[dst] = append(destinations[dst], ev)
	}
	return events, destinations
}

func verifySpecifiedTypes(types []string) {
	for _, t := range types {
		if !strings.HasSuffix(t, declarationTag) || len(t) < 12 {
			log.Fatalf("unsuported type name. type name should have `Declaration` suffix. type: %s", t)
		}
	}
}
func verifyDefinition(fset *token.FileSet, definition MachineDefinition) {
	for _, st := range definition.States {
		for dst, events := range st.Destinations {
			_, ok := definition.States[dst]
			if !ok {
				log.Fatalf(
					"You've defined (%v) -%v-> (%v). But there is no such destination State as `%v`. %v",
					st.Name, events, dst, dst, fset.Position(st.Field.Pos()),
				)
			}
		}
	}
}

func verifyField(fset *token.FileSet, field *ast.Field) {
	typeID, ok := field.Type.(*ast.Ident)
	if !ok || typeID.Name != "FSMState" {
		log.Fatalf("target field type unsupported %+v. %v", field.Type, fset.Position(field.Pos()))
	}
	if len(field.Names) != 1 {
		log.Fatalf("target field names have unexpected len: %+v. %v", field.Names, fset.Position(field.Pos()))
	}
}

func extractStructTypeFromDefinition(fset *token.FileSet, obj *ast.Object) *ast.StructType {
	if obj.Kind != ast.Typ {
		log.Fatalf("target type kind unsuported %+v. %v", obj, fset.Position(obj.Pos()))
	}
	typeSpec, ok := obj.Decl.(*ast.TypeSpec)
	if !ok {
		log.Fatalf("target type declaration unsuported %+v. %v", obj.Decl, fset.Position(obj.Pos()))
	}
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		log.Fatalf("type specification is not struct type %+v. %v", typeSpec, fset.Position(typeSpec.Pos()))
	}
	if structType.Incomplete || structType.Fields == nil || len(structType.Fields.List) == 0 {
		log.Fatalf("target struct is incoplete or has zero fields %+v. %v", typeSpec, fset.Position(typeSpec.Pos()))
	}
	return structType
}

func strip(s string) string {
	return s[1 : len(s)-1]
}
