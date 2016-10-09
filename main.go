package main

import (
	"flag"
	"io"
	"os"
	"path"
	"strings"
	"text/template"
)

type Node struct {
	IsFolder bool
	Dynamic  bool
	Name     string
	Prev     *Node
	Target   string
}

func (n *Node) NextFolder(name string) *Node {
	v := &Node{IsFolder: true, Name: name, Prev: n}
	return v
}

func (n *Node) NextFile(name string) *Node {
	v := &Node{IsFolder: false, Name: name, Prev: n}
	return v
}
func (n *Node) Class() string {
	if n.IsFolder {
		return strings.Title(n.Name) + "Collection"
	}
	return strings.Title(n.Name) + "Record"
}
func (n *Node) Title() string {
	return strings.Title(n.Name)
}

func (n *Node) Param() string {
	return strings.ToLower(n.Name)
}

func (n *Node) Path() string {
	v := ""
	if n.Prev != nil {
		v += n.Prev.Path() + "/"
	}
	return v + n.Name
}

const fileT = `
type {{.Class}} struct {
	path    string
	Id      string
	Content []byte
	ModTime time.Time
}

{{if .Target}}
func (cls {{.Class}}) {{.Target}}() ({{.Target}}, error) {
	var target {{.Target}}
	return target, json.Unmarshal(cls.Content, &target)
}
{{end}}

func (cls {{.Class}}) Get(id string, target interface{}) error  {
	return json.Unmarshal(cls.Content, target)
}

func (cls {{.Class}}) Location() string  {
	return cls.path
}

func (cls {{.Class}}) Set(obj interface{}) error {
	data, err := json.MarshalIndent(obj, "    ", "")
	if err !=nil {
		return err
	}
	cls.Content = data
	return nil
}

func (cls {{.Class}}) Save() error {
	err := os.MkdirAll(path.Dir(cls.path), 0755)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cls.path, cls.Content, 0755)
}

{{if .Target}}
func (container {{.Prev.Class}}) Get{{.Title}}(id string) ({{.Target}}, error)  {
	var target {{.Target}}
	data, err := ioutil.ReadFile(path.Join(container.location, id))
	if err!=nil{
		return target, err
	}
	return target, json.Unmarshal(data, &target)
}
{{end}}

func (container {{.Prev.Class}}) Get{{.Class}}(id string) ({{.Class}}, error) {
	var result {{.Class}}
	file := path.Join(container.location, id)
	info, err := os.Stat(file)
	if err!=nil {
		return result, err
	}
	result.ModTime = info.ModTime()
	result.path = file
	data, err := ioutil.ReadFile(result.path)
	if err!=nil{
		return result, err
	}
	result.Content = data	
	return result, nil
}

func (container {{.Prev.Class}}) Save{{.Title}}Binary(id string, content []byte) error {
	err := os.MkdirAll(container.location, 0755)
	if err != nil {
		return err
	}
	return ioutil.WriteFile( path.Join(container.location, id), content, 0755)
}

func (container {{.Prev.Class}}) Save{{.Title}}(id string, content interface{}) error {
	data, err := json.MarshalIndent(content, "    ", "")
	if err !=nil {
		return err
	}
	return container.Save{{.Title}}Binary(id, data)
}


func (container {{.Prev.Class}}) Iterate{{.Title}}s(handler func({{.Class}}) error) error {
	info, err := ioutil.ReadDir(container.location)
	if err != nil {
		return err
	}
	for _, inf := range info {
		if !inf.IsDir() {
			data, err := ioutil.ReadFile(path.Join(container.location, inf.Name()))
			if err!=nil{
				return err
			}
			err = handler( {{.Class}}{Id:inf.Name(), Content: data, ModTime: inf.ModTime()})
			if err!=nil{
				return err
			}
		}
	}
	return nil
}

func (container {{.Prev.Class}}) List{{.Title}}s() (items []string) {
	info, err := ioutil.ReadDir(container.location)
	if err != nil {
		log.Println("failed scan {{.Title}} in collection {{.Prev.Title}}", container.location, err)
		return
	}
	for _, inf := range info {
		if !inf.IsDir() {
			items = append(items, inf.Name())
		}
	}
	return
}
`

const headerT = `
package {{.Package}}

import (
	"path"
	{{if or .Files .Dynamic}}
	"io/ioutil"
	{{end}}
	{{if .Files}}
	"os"
	"log"
	"time"
	"encoding/json"
	{{end}}
)
`

const dirT = `
type {{.Class}} struct {
	location string
}

{{if .Dynamic}}
func (container {{.Prev.Class}}) {{.Title}}(id string) {{.Class}} {
	return {{.Class}}{location: path.Join(container.location, id)}
}

func (container {{.Prev.Class}}) List{{.Title}}s() (items []string) {
	info, err := ioutil.ReadDir(container.location)
	if err != nil {
		log.Println("failed scan {{.Title}} in collection {{.Prev.Title}}", container.location, err)
		return
	}
	for _, inf := range info {
		if inf.IsDir() {
			items = append(items, inf.Name())
		}
	}
	return
}
{{else if .Prev}}
func (container {{.Prev.Class}}) {{.Title}}() {{.Class}} {
	return {{.Class}}{location: path.Join(container.location, "{{.Param}}")}
}
{{else}}
func New{{.Title}}(rootLocation string) {{.Class}} {
	return {{.Class}} {location: rootLocation}
}
{{end}}

`

func create(root *Node, out io.Writer, done *map[string]bool) {
	templ, err := template.New("").Parse(fileT)
	if err != nil {
		panic(err)
	}
	templ2, err := template.New("").Parse(dirT)
	if err != nil {
		panic(err)
	}
	templ3, err := template.New("").Parse(headerT)
	if err != nil {
		panic(err)
	}

	var s = root
	var file bool
	var dyn bool
	for s != nil {
		if !s.IsFolder {
			file = true
		}
		if s.Dynamic {
			dyn = true
		}
		s = s.Prev
	}

	err = templ3.Execute(out, map[string]interface{}{"Files": file, "Dynamic": dyn, "Package": *packageName})
	if err != nil {
		panic(err)
	}
	var p = root
	for p != nil {
		if !(*done)[p.Path()] {
			if p.IsFolder {
				err = templ2.Execute(out, p)
			} else {
				err = templ.Execute(out, p)
			}
			if err != nil {
				panic(err)
			}
			(*done)[p.Path()] = true
			p = p.Prev
		} else {
			break
		}
	}
}

var (
	packageName = flag.String("p", "dummy", "Package name")
	rootFolder  = flag.String("r", "data", "Root folder name")
	outFolder   = flag.String("o", "./dummy", "Output folder name")
)

/*
Grammar:

(/[:]SECTION)+ [-> CLASS]

Example:

/:group/users/user -> User
/:group/meta/meta -> Meta
*/
func parseLink(root *Node, url string) *Node {

	var target string
	var path string
	parts := strings.SplitN(url, "->", 2)
	if len(parts) == 2 {
		target = strings.TrimSpace(parts[1])
	}
	path = strings.TrimSpace(parts[0])
	sections := strings.Split(path, "/")
	n := root

	var N = len(sections)
	for i, section := range sections {
		if len(section) == 0 {
			continue
		}
		var dyno bool
		if strings.HasPrefix(section, ":") {
			dyno = true
			section = section[1:]
		}
		if i == N-1 {
			n = n.NextFile(section)
			n.Target = target
		} else {
			n = n.NextFolder(section)
			n.Dynamic = dyno
		}
	}
	return n
}

func main() {
	flag.Parse()
	root := &Node{Name: *rootFolder, IsFolder: true}
	var roots []*Node
	for _, link := range flag.Args() {
		roots = append(roots, parseLink(root, link))
	}

	done := make(map[string]bool)
	if err := os.MkdirAll(*outFolder, 0755); err != nil {
		panic(err)
	}
	for _, node := range roots {
		func(node *Node) {
			file := "fsdb_" + strings.ToLower(node.Name) + "_record" + ".go"
			if node.Target != "" {
				file = "fsdb_" + strings.ToLower(node.Target) + ".go"
			}
			f, err := os.OpenFile(path.Join(*outFolder, file), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			create(node, f, &done)
		}(node)

	}

	//	db:=NewRoot("")

}
