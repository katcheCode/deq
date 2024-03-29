//go:generate go-bindata -pkg main -o tpl.go out.go.tpl

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	// "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	proto "github.com/gogo/protobuf/proto"
	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
)

type File struct {
	Name     string
	Source   string
	Package  string
	Services []Service
	Types    []Type
}

type Type struct {
	Name string
}

type Service struct {
	Name    string
	Methods []Method
	File    File
	Types   []Type
}

type Method struct {
	Name    string
	InType  string
	OutType string
	Service Service
}

func main() {

	input := new(plugin.CodeGeneratorRequest)
	response := new(plugin.CodeGeneratorResponse)

	inbuf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		response.Error = proto.String(fmt.Sprintf("read input: %v", err))
		reply(response)
	}

	err = proto.Unmarshal(inbuf, input)
	if err != nil {
		response.Error = proto.String(fmt.Sprintf("unmarshal input: %v", err))
		reply(response)
	}

	files, err := generate(input)
	if err != nil {
		response.Error = proto.String(err.Error())
	} else {
		response.File = files
	}

	reply(response)
}

func reply(response *plugin.CodeGeneratorResponse) {
	outBuf, err := proto.Marshal(response)
	if err != nil {
		panic("marshal response: " + err.Error())
	}
	os.Stdout.Write(outBuf)
	if response.Error != nil {
		log.Printf("%s", *response.Error)
		os.Exit(1)
	}
}

func generate(input *plugin.CodeGeneratorRequest) ([]*plugin.CodeGeneratorResponse_File, error) {

	outfiles := make([]*plugin.CodeGeneratorResponse_File, len(input.GetProtoFile()))

	for i, descriptor := range input.GetProtoFile() {

		if len(descriptor.GetMessageType()) == 0 {
			continue
		}

		dir := filepath.Dir(descriptor.GetName())
		baseFileName := filepath.Base(descriptor.GetName())
		baseName := strings.TrimSuffix(baseFileName, filepath.Ext(baseFileName))

		pkg := descriptor.GetOptions().GetGoPackage()
		if pkg == "" {
			pkg = baseName
		}
		lastSep := strings.LastIndex(pkg, "/")
		if lastSep != -1 {
			pkg = pkg[lastSep+1:]
		}

		file := File{
			Name:     filepath.Join(input.GetParameter(), dir, baseName+".pb.deq.go"),
			Source:   descriptor.GetName(),
			Package:  pkg,
			Services: make([]Service, len(descriptor.Service)),
			Types:    make([]Type, len(descriptor.MessageType)),
		}

		for j, mType := range descriptor.GetMessageType() {
			file.Types[j] = Type{
				Name: goName(mType.GetName()),
			}
		}

		for j, svc := range descriptor.GetService() {

			methods := make([]Method, len(svc.GetMethod()))
			typeSet := make(map[Type]struct{})

			for k, method := range svc.GetMethod() {
				inType := goName(method.GetInputType())
				outType := goName(method.GetOutputType())
				methods[k] = Method{
					Name:    method.GetName(),
					InType:  inType,
					OutType: outType,
					Service: file.Services[j],
				}

				typeSet[Type{
					Name: inType,
				}] = struct{}{}

				typeSet[Type{
					Name: outType,
				}] = struct{}{}
			}

			types := make([]Type, len(typeSet))
			var i int
			for t := range typeSet {
				types[i] = t
				i++
			}

			file.Services[j] = Service{
				Name:    svc.GetName(),
				File:    file,
				Methods: methods,
				Types:   types,
			}
		}

		outfile := &plugin.CodeGeneratorResponse_File{
			Name: &file.Name,
		}

		data, err := Asset("out.go.tpl")
		if err != nil {
			return nil, fmt.Errorf("load template: %v", err)
		}

		tpl, err := template.New("main").Parse(string(data))
		if err != nil {
			return nil, fmt.Errorf("parse template: %v", err)
		}

		outbuffer := new(bytes.Buffer)
		err = tpl.Execute(outbuffer, file)
		if err != nil {
			return nil, fmt.Errorf("run template: %v", err)
		}
		outfile.Content = proto.String(outbuffer.String())

		outfiles[i] = outfile
	}

	return outfiles, nil
}

// CamelCase returns the CamelCased name.
// If there is an interior underscore followed by a lower case letter,
// drop the underscore and convert the letter to upper case.
// There is a remote possibility of this rewrite causing a name collision,
// but it's so remote we're prepared to pretend it's nonexistent - since the
// C++ generator lowercases names, it's extremely unlikely to have two fields
// with different capitalizations.
// In short, _my_field_name_2 becomes XMyFieldName_2.
func CamelCase(s string) string {
	if s == "" {
		return ""
	}
	t := make([]byte, 0, 32)
	i := 0
	if s[0] == '_' {
		// Need a capital letter; drop the '_'.
		t = append(t, 'X')
		i++
	}
	// Invariant: if the next letter is lower case, it must be converted
	// to upper case.
	// That is, we process a word at a time, where words are marked by _ or
	// upper case letter. Digits are treated as words.
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' && i+1 < len(s) && isASCIILower(s[i+1]) {
			continue // Skip the underscore in s.
		}
		if isASCIIDigit(c) {
			t = append(t, c)
			continue
		}
		// Assume we have a letter now - if not, it's a bogus identifier.
		// The next word is a sequence of characters that must start upper case.
		if isASCIILower(c) {
			c ^= ' ' // Make it a capital letter.
		}
		t = append(t, c) // Guaranteed not lower case.
		// Accept lower case sequence that follows.
		for i+1 < len(s) && isASCIILower(s[i+1]) {
			i++
			t = append(t, s[i])
		}
	}
	return string(t)
}

// Is c an ASCII lower-case letter?
func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

// Is c an ASCII digit?
func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func LastComponent(s, sep string) string {
	parsed := strings.Split(s, sep)
	return parsed[len(parsed)-1]
}

func goName(name string) string {
	return CamelCase(LastComponent(name, "."))
}
