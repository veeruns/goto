package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"html/template"
	"reflect"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	flag "github.com/spf13/pflag"
)

//Shortcut struct for keeping shortcut structure
type Shortcut struct {
	ID  string
	URL string
}

var templateFuncs = template.FuncMap{"rangeStruct": RangeStructer}
var shortcuts []Shortcut
var db *bolt.DB

func main() {
	var port int
	flag.IntVar(&port, "port", 8080, "Set port to listen")
	flag.Parse()
	r := mux.NewRouter()
	db, _ = setupDB()

	r.HandleFunc("/goto", TemplateHandler)

	r.HandleFunc("/testurl", TestURLHandler)

	r.HandleFunc("/addentry", AddEntryHandler).Methods("POST")

	r.HandleFunc("/{category}", HomeHandler).Methods("GET")

	r.HandleFunc("/css/{file}", AddStaticHTML).Methods("GET")

	r.HandleFunc("/{section}/{category}", SectionHandler).Methods("GET")

	r.PathPrefix("/css/").Handler(http.StripPrefix("/css/", http.FileServer(http.Dir("./css/"))))

	r.HandleFunc("/", firstpage)

	http.Handle("/", r)
	listenaddress := fmt.Sprintf("localhost:%d", port)
	http.ListenAndServe(listenaddress, r)
}

//TestURLHandler is the test urlhandler for http parameters
func TestURLHandler(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Query()
	for k, v := range u {
		fmt.Fprintf(w, "%v and %v\n", k, v)
	}
	fmt.Fprintf(w, "%v\n", u)
}

//TemplateHandler is the function that handles the table css template
func TemplateHandler(w http.ResponseWriter, r *http.Request) {
	/*var htmlTemplate = `{{range .}}<tr>
	{{range rangeStruct .}}	<td>{{.}}</td>
	{{end}}</tr>
	{{end}}`*/
	var shcuts []Shortcut
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("SHORTCUT"))
		b.ForEach(func(k, v []byte) error {
			var s Shortcut

			if v != nil {
				s.ID = string(k)
				s.URL = string(v)
			} else {
				s.ID = string(k)
				s.URL = "Bucket"
			}

			shcuts = append(shcuts, s)
			return nil
		})

		return nil
	})
	if err != nil {
		fmt.Printf("Unable to read the db %v\n", err.Error())
	}
	tf, err := template.New("mainpage.gohtml").Funcs(templateFuncs).ParseFiles("mainpage.gohtml")
	if err != nil {
		fmt.Printf("Error is %v\n", err.Error())

	}

	err = tf.Execute(w, shcuts)
	if err != nil {
		fmt.Printf("Error is %s\n", err.Error())
		fmt.Fprintf(w, "Unable to open Add entry handler: %s\n", err.Error())
	}
}

//AddStaticHTML is the section for static html
func AddStaticHTML(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	file := vars["file"]
	filetoopen := fmt.Sprintf("css/%s", file)
	data, err := ioutil.ReadFile(filetoopen)
	if err != nil {
		http.Error(w, "Couldn't read file", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

//AddEntryHandler is the section that redirects to the section
func AddEntryHandler(w http.ResponseWriter, r *http.Request) {
	shortcut := r.FormValue("shortcut")
	section := r.FormValue("section")
	url := r.FormValue("url")
	if len(section) > 0 {
		if len(shortcut) > 0 && len(url) > 0 {
			addEntrySection(db, section, shortcut, url)
			w.Header().Set("location", "/goto")
			http.Redirect(w, r, "/goto", 301)
		} else {
			fmt.Fprintf(w, "Cannot give empty value")
			w.WriteHeader(http.StatusBadRequest)
		}
	} else {
		if len(shortcut) > 0 && len(url) > 0 {
			addEntry(db, shortcut, url)
			w.Header().Set("location", "/goto")
			http.Redirect(w, r, "/goto", 301)
		} else {
			fmt.Fprintf(w, "Cannot give empty value")
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}

//HomeHandler home handler definition
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	redirecturl, _ := getEntry(db, vars["category"])
	if redirecturl == "" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Not found in Database %v\n", vars["category"])

	} else {
		http.Redirect(w, r, redirecturl, 301)
	}

	fmt.Fprintf(w, "Category: %v\n", vars["category"])
}

//SectionHandler home handler definition
func SectionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	redirecturl, _ := getEntrySection(db, vars["section"], vars["category"])
	if redirecturl == "" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Not found in Database %v\n", vars["category"])

	} else {
		http.Redirect(w, r, redirecturl, 301)
	}

	fmt.Fprintf(w, "Category: %v\n", vars["category"])
}

func firstpage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "First page ")

}

func setupDB() (*bolt.DB, error) {
	db, err := bolt.Open("goto.db", 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("could not open db, %v", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("SHORTCUT"))
		if err != nil {
			return fmt.Errorf("could not create root bucket: %v", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not set up buckets, %v", err)
	}
	fmt.Println("DB Setup Done")
	return db, nil
}
func createSection(db *bolt.DB, section string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists([]byte("SHORTCUT"))
		if err != nil {
			return fmt.Errorf("Could not create root bucket while creating section: %v", err)
		}
		_, err = root.CreateBucketIfNotExists([]byte(section))
		if err != nil {
			return fmt.Errorf("Create section failed: %v", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Update DB failed : %v", err)
	}
	return nil
}
func addEntry(db *bolt.DB, id string, url string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("SHORTCUT")).Put([]byte(id), []byte(url))
		if err != nil {
			return fmt.Errorf("Could not add shortcut: %v", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Could not update database: %v", err)

	}
	fmt.Printf("Database updated\n")
	return err
}
func addEntrySection(db *bolt.DB, section string, id string, url string) error {
	err := createSection(db, section)
	if err != nil {
		return fmt.Errorf("Error creating section addEntrySection: %v", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("SHORTCUT")).Bucket([]byte(section)).Put([]byte(id), []byte(url))
		if err != nil {
			return fmt.Errorf("Could not add shortcut: %v", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Could not update database: %v", err)

	}
	fmt.Printf("Database updated\n")
	return err
}

func getEntry(db *bolt.DB, id string) (string, error) {
	var url []byte
	err := db.View(func(tx *bolt.Tx) error {
		url = tx.Bucket([]byte("SHORTCUT")).Get([]byte(id))
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("Unable to getEntry: %v", err)
	}

	return string(url), nil
}

func getEntrySection(db *bolt.DB, section string, id string) (string, error) {
	var url []byte
	fmt.Printf("Section is %s\n", section)
	err := db.View(func(tx *bolt.Tx) error {
		url = tx.Bucket([]byte("SHORTCUT")).Bucket([]byte(section)).Get([]byte(id))
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("Unable to getEntry: %v", err)
	}

	return string(url), nil
}

// RangeStructer takes the first argument, which must be a struct, and
// returns the value of each field in a slice. It will return nil
// if there are no arguments or first argument is not a struct
func RangeStructer(args ...interface{}) []interface{} {
	if len(args) == 0 {
		return nil
	}

	v := reflect.ValueOf(args[0])
	fmt.Printf("Kind is %v\n", v.Kind())
	if v.Kind() != reflect.Struct {
		return nil
	}

	out := make([]interface{}, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		out[i] = v.Field(i).Interface()

	}

	return out

}
