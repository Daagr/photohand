package photohand

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	goji "goji.io"
	"goji.io/pat"
)

func errr(w http.ResponseWriter, r *http.Request, err error) {
	fmt.Fprintln(w, "TODO: error handling")
	log.Println(err)
}

func Serve(host, path string) {
	mux := goji.NewMux()
	app := goji.SubMux()

	pathst := fmt.Sprint(path, "/*")
	if strings.HasSuffix(path, "/") {
		pathst = fmt.Sprint(path, "*")
	}
	mux.Handle(pat.New(pathst), app)

	app.HandleFunc(pat.Get("/"), func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "spa.html")
	})
	app.HandleFunc(pat.Get("/vue.js"), func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "vue.js")
	})
	app.HandleFunc(pat.Get("/axios.min.js"), func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "axios.min.js")
	})

	app.HandleFunc(pat.Get("/app.js"), func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "app.js")
	})

	app.HandleFunc(pat.Get("/update"), func(w http.ResponseWriter, r *http.Request) {
		Update()
		fmt.Fprintln(w, "Done")
	})

	app.HandleFunc(pat.Get("/list"), func(w http.ResponseWriter, r *http.Request) {
		var query strings.Builder
		query.WriteString("SELECT * FROM photos WHERE 1 = 1 ")
		queryargs := make([]interface{}, 0)
		formnum := func(name string) {
			v := r.FormValue(name)
			if v == "" {
				return
			}
			f64, err := strconv.ParseFloat(strings.TrimLeft(v, "=<>"), 32)
			if err != nil {
				errr(w, r, err)
			}
			f := float32(f64)
			lt := v[0] == '<'
			gt := v[0] == '>'
			query.WriteString(" AND ")
			if lt {
				query.WriteString(name)
				query.WriteString(" <= ? AND ")
				query.WriteString(name)
				query.WriteString(" != 0 ")
				queryargs = append(queryargs, f)
			} else if gt {
				query.WriteString(name)
				query.WriteString(" >= ? ")
				queryargs = append(queryargs, f)
			} else {
				query.WriteString(name)
				query.WriteString(" = ? ")
				queryargs = append(queryargs, f)
			}
		}
		formnum("f")
		formnum("time")
		formnum("iso")
		formnum("exposurebias")
		formnum("focallength")
		formnum("width")
		formnum("height")
		formnum("rating")
		// phs, err := QueryPhs("SELECT * FROM photos")
		log.Println(query.String())
		log.Println(queryargs...)
		phs, err := QueryPhs(query.String(), queryargs...)
		if err != nil {
			errr(w, r, err)
			return
		}

		b, err := json.Marshal(phs)
		if err != nil {
			errr(w, r, err)
			return
		}

		w.Header().Set("Content-Type", "text/json")
		_, err = w.Write(b)
		if err != nil {
			errr(w, r, err)
			return
		}
	})

	app.HandleFunc(pat.Get("/rate/:id/:num"), func(w http.ResponseWriter, r *http.Request) {
		uuid := pat.Param(r, "id")
		num, err := strconv.Atoi(pat.Param(r, "num"))
		if err != nil {
			errr(w, r, err)
			return
		}
		_, err = db.Exec("UPDATE photos SET rating = ? WHERE uuid = ?", num, uuid)
		if err != nil {
			errr(w, r, err)
		}
		fmt.Fprintln(w, "{}")
	})

	app.HandleFunc(pat.Get("/thumb/:id"), func(w http.ResponseWriter, r *http.Request) {
		uuid := pat.Param(r, "id")
		filename := fmt.Sprint(filepath.Join(datadir, "/thumbs/", uuid), ".jpg")
		file, err := os.Open(filename)
		if os.IsNotExist(err) {
			err = CreateThumb(uuid, filename)
			if err != nil {
				errr(w, r, err)
				return
			}
			file, err = os.Open(filename)
		}
		if err != nil {
			errr(w, r, err)
			return
		}

		file.Close()
		http.ServeFile(w, r, filename)
	})

	app.HandleFunc(pat.Get("/mid/:id"), func(w http.ResponseWriter, r *http.Request) {
		uuid := pat.Param(r, "id")
		filename := fmt.Sprint(filepath.Join(datadir, "/mids/", uuid), ".jpg")
		file, err := os.Open(filename)
		if os.IsNotExist(err) {
			err = CreateMid(uuid, filename)
			if err != nil {
				errr(w, r, err)
				return
			}
			file, err = os.Open(filename)
		}
		if err != nil {
			errr(w, r, err)
			return
		}
		file.Close()
		http.ServeFile(w, r, filename)
	})

	app.HandleFunc(pat.Get("/full/:id"), func(w http.ResponseWriter, r *http.Request) {
		uuid := pat.Param(r, "id")
		ph, err := QueryPh("SELECT * FROM photos WHERE uuid = ?", uuid)
		if err != nil {
			errr(w, r, err)
			return
		}
		http.ServeFile(w, r, ph.Path)
	})

	log.Printf("Serving at %s%s", host, pathst)
	log.Fatal(http.ListenAndServe(host, mux))
}
