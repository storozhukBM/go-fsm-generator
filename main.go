package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

const declarationTag = "Declaration"

type event string
type state string

type stateDefinition struct {
	Name         state
	Events       map[event]state
	Destinations map[state][]event
	IsTerminal   bool
	Field        *ast.Field
}

type machineDefinition struct {
	DirName     string
	PkgName     string
	MachineName string
	States      map[state]stateDefinition
	Description string
	Struct      *ast.StructType
}

func main() {
	verbose := flag.Bool("v", false, "verbose output from generator")
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
	scan(pkgs, types, func(pkg *ast.Package, foundType string, obj *ast.Object) {

		machineName := strings.TrimSuffix(foundType, declarationTag)
		generateStm(*verbose, machineName, dirName, pkg.Name, fset, obj)
	})
}

func generateStm(verbose bool, machineName string, dirName string, pkgName string, fset *token.FileSet, obj *ast.Object) {
	structType := extractStructTypeFromDefinition(fset, obj)
	states := map[state]stateDefinition{}
	for _, field := range structType.Fields.List {
		verifyField(fset, field)
		st := stateDefinition{
			Name:       state(field.Names[0].Name),
			IsTerminal: field.Tag == nil || field.Tag.Value == "",
			Field:      field,
		}
		st.Events, st.Destinations = parseStateMachineEventsAndDestinations(st, fset, field.Tag)
		states[st.Name] = st
	}
	definition := machineDefinition{
		DirName:     dirName,
		PkgName:     pkgName,
		MachineName: machineName,
		States:      states,
		Struct:      structType,
	}
	verifyDefinition(fset, definition)
	definition.Description = describeGeneratedMachine(definition)
	generateFromTemplateAndWriteToFile(definition)
	if verbose {
		fmt.Println(strip(definition.Description))
	}
}

func generateFromTemplateAndWriteToFile(definition machineDefinition) {
	var b bytes.Buffer
	err := embeddedTemplate.Execute(&b, definition)
	if err != nil {
		log.Fatal("can't execute template ", err)
	}
	src, err := format.Source(b.Bytes())
	if err != nil {
		log.Fatal("can't format generated template ", err)
	}
	output := strings.ToLower(definition.MachineName + ".fsm.go")
	absPath, err := filepath.Abs(definition.DirName)
	if err != nil {
		log.Fatal("can't calculate abs path for: "+definition.DirName, err)
	}
	outputPath := filepath.Join(absPath, output)
	log.Print(outputPath)
	err = ioutil.WriteFile(outputPath, src, 0664)
	if err != nil {
		log.Fatal("can't write file to disk. ", err)
	}
}

func describeGeneratedMachine(definition machineDefinition) string {
	builder := &strings.Builder{}

	builder.WriteString("`// Definition for ")
	builder.WriteString(definition.MachineName)
	builder.WriteString(" in Graphviz format \n")
	builder.WriteString("digraph ")
	builder.WriteString(definition.MachineName)
	builder.WriteString(" {\n")

	for _, state := range sortedStates(definition.States) {
		stateDef := definition.States[state]
		if stateDef.IsTerminal {
			builder.WriteString("	")
			builder.WriteString(string(state))
			builder.WriteString(" [shape=Msquare];\n")
			continue
		}

		for _, ev := range sortedEvents(stateDef.Events) {
			dst := stateDef.Events[ev]
			builder.WriteString("	")
			builder.WriteString(string(state))
			builder.WriteString(" -> ")
			builder.WriteString(string(dst))
			builder.WriteString(" [label=")
			builder.WriteString(string(ev))
			builder.WriteString("];\n")
		}
	}
	builder.WriteString("}\n`")

	return builder.String()
}

func parseStateMachineEventsAndDestinations(st stateDefinition, fset *token.FileSet, tag *ast.BasicLit) (map[event]state, map[state][]event) {
	if st.IsTerminal {
		return nil, nil
	}
	events := map[event]state{}
	destinations := map[state][]event{}
	eventsDeclarations := strings.Split(strip(tag.Value), ",")
	for _, eventDeclaration := range eventsDeclarations {
		eventStr := strings.Split(eventDeclaration, ":")
		if len(eventStr) != 2 || len(eventStr[0]) < 1 || len(eventStr[0]) < 3 {
			log.Fatalf("unsuported tag format %+v. %v", eventDeclaration, fset.Position(tag.Pos()))
		}
		ev := event(eventStr[0])
		dst := state(strip(eventStr[1]))

		if ev == "Noop" {
			log.Fatalf("event `Noop` is reserved by system %+v", fset.Position(tag.Pos()))
		}

		if _, ok := events[ev]; ok {
			log.Fatalf("event `%s` duplicate on state `%s`. %v", ev, st.Name, fset.Position(tag.Pos()))
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

func verifyDefinition(fset *token.FileSet, definition machineDefinition) {
	for _, st := range definition.States {
		for dst, events := range st.Destinations {
			_, ok := definition.States[dst]
			if !ok {
				log.Fatalf(
					"You've defined (%v) -%v-> (%v). But there is no such destination state as `%v`. %v",
					st.Name, events, dst, dst, fset.Position(st.Field.Pos()),
				)
			}
		}
	}
}

func verifyField(fset *token.FileSet, field *ast.Field) {
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

func scan(
	packages map[string]*ast.Package,
	lookingForTypes []string,
	apply func(pkg *ast.Package, foundType string, obj *ast.Object),
) {
	for _, pkg := range packages {
		for _, file := range pkg.Files {
			for _, t := range lookingForTypes {
				obj := file.Scope.Lookup(t)
				if obj == nil {
					continue
				}
				apply(pkg, t, obj)
			}
		}
	}
}

func sortedStates(m map[state]stateDefinition) []state {
	var result []state
	for key := range m {
		result = append(result, key)
	}
	sort.Slice(result, func(i, j int) bool {
		return string(result[i]) < string(result[j])
	})
	return result
}

func sortedEvents(m map[event]state) []event {
	var result []event
	for key := range m {
		result = append(result, key)
	}
	sort.Slice(result, func(i, j int) bool {
		return string(result[i]) < string(result[j])
	})
	return result
}
