package main

import (
	"github.com/joho/godotenv"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type pageData struct {
	Title   string
	Content interface{}
}

type pageModel struct {
	Title string
	Body  []byte
}

type indexData struct {
	Items []string
}

var validPath = regexp.MustCompile("^(?:/|/(view|edit|save|delete)/([a-zA-Z0-9]+))$")

var templates = template.Must(template.ParseGlob("templates/*.html"))

func indexHandler(w http.ResponseWriter, r *http.Request, param string) {
	pattern := filepath.Join(os.Getenv("STORAGE_PATH"), "*.txt")
	files, err := filepath.Glob(pattern)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i, file := range files {
		files[i] = strings.TrimSuffix(strings.TrimPrefix(file, os.Getenv("STORAGE_PATH")+"/"), ".txt")
	}

	data := pageData{
		Title: "All Pages",
		Content: &indexData{
			files,
		},
	}

	renderTemplate(w, data, "index")
}

func viewHandler(w http.ResponseWriter, r *http.Request, param string) {
	p, err := loadPage(param)
	if err != nil {
		http.Redirect(w, r, "/edit/"+param, http.StatusFound)
		return
	}

	data := pageData{
		Title:   "View " + param,
		Content: p,
	}

	renderTemplate(w, data, "view")
}

func saveHandler(w http.ResponseWriter, r *http.Request, param string) {
	body := r.FormValue("body")
	title := r.FormValue("title")
	p := &pageModel{Title: title, Body: []byte(body)}

	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func deleteHandler(w http.ResponseWriter, r *http.Request, param string) {
	p, err := loadPage(param)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = p.delete()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func editHandler(w http.ResponseWriter, r *http.Request, param string) {
	p, err := loadPage(param)
	if err != nil {
		p = &pageModel{Title: param}
	}

	data := pageData{
		Title:   "Edit " + param,
		Content: p,
	}

	renderTemplate(w, data, "edit")
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}

		fn(w, r, m[2])
	}
}

func renderTemplate(w http.ResponseWriter, pageData pageData, tmpl string) {
	baseTmpl := templates.Lookup("base.html")
	contentTmpl := templates.Lookup(tmpl + ".html")

	if baseTmpl == nil || contentTmpl == nil {
		http.Error(w, "Not found base or content template", http.StatusInternalServerError)
		return
	}

	var contentBuf strings.Builder
	err := contentTmpl.Execute(&contentBuf, pageData.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseData := struct {
		Title   string
		Content template.HTML
	}{
		Title:   pageData.Title,
		Content: template.HTML(contentBuf.String()),
	}

	err = templates.ExecuteTemplate(w, "base.html", baseData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (p *pageModel) save() error {
	filename := os.Getenv("STORAGE_PATH") + "/" + p.Title + ".txt"

	if _, err := os.Stat(os.Getenv("STORAGE_PATH")); os.IsNotExist(err) {
		err := os.Mkdir(os.Getenv("STORAGE_PATH"), 0750)
		if err != nil {
			return err
		}
	}

	return os.WriteFile(filename, p.Body, 0600)
}

func (p *pageModel) delete() error {
	filename := os.Getenv("STORAGE_PATH") + "/" + p.Title + ".txt"

	if _, err := os.Stat(os.Getenv("STORAGE_PATH")); os.IsNotExist(err) {
		err := os.Mkdir(os.Getenv("STORAGE_PATH"), 0750)
		if err != nil {
			return err
		}
	}

	return os.Remove(filename)
}

func loadPage(param string) (*pageModel, error) {
	fn := os.Getenv("STORAGE_PATH") + "/" + param + ".txt"

	body, err := os.ReadFile(fn)
	if err != nil {
		return nil, err
	}

	return &pageModel{Title: param, Body: body}, nil
}

func setupEnv() {
	if err := godotenv.Load(); err != nil {
		slog.Error("error initializing env variables:", err.Error())
	}
}

func main() {
	setupEnv()

	http.HandleFunc("/", makeHandler(indexHandler))
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/delete/", makeHandler(deleteHandler))

	log.Println("Server starting on this address: http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Ошибка сервера:", err)
	}
}
