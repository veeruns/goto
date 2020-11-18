package main

import (
	"fmt"
	"net/http"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

type Shortcut struct {
	Id  string
	Url string
}

var db *bolt.DB

func main() {
	r := mux.NewRouter()
	db, _ = setupDB()
	addEntry(db, "ya", "https://www.yahoo.com")
	addEntry(db, "go", "https://www.google.com")
	r.HandleFunc("/{category}", HomeHandler)

	http.Handlefunc("/", firstpage)
	http.ListenAndServe(":8080", r)
}

//HomeHandler home handler definition
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	redirecturl, _ := getEntry(db, vars["category"])
	if redirecturl == nil {
		fmt.Fprintf(w, "Not found in Database %v\n", vars["category"])
		w.WriteHeader(http.StatusOK)
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
