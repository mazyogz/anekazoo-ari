package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type Animal struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Class string `json:"class"`
    Legs  int    `json:"legs"`
}

var db *sql.DB

func testDBConnection() {
    err := db.Ping()
    if err != nil {
        log.Println("Database connection failed:", err)
    } else {
        log.Println("Database connected successfully")
    }
}

func migrateDB() {
    query := `
    CREATE TABLE IF NOT EXISTS animals (
        id SERIAL PRIMARY KEY,
        name VARCHAR(50) NOT NULL UNIQUE,
        class VARCHAR(50) NOT NULL,
        legs INT NOT NULL
    );`

    _, err := db.Exec(query)
    if err != nil {
        log.Fatalf("Error creating table: %v\n", err)
    }

    var tableExists bool
    err = db.QueryRow(`SELECT EXISTS (
        SELECT 1
        FROM   information_schema.tables 
        WHERE  table_schema = 'public'
        AND    table_name = 'animals'
    );`).Scan(&tableExists)

    if err != nil {
        log.Fatalf("Error checking if table exists: %v\n", err)
    }

    if tableExists {
        log.Println("Models already migrated.")
    } else {
        log.Println("Migration completed successfully.")
    }
}
func main() {
    var err error
    db, err = sql.Open("postgres", "host=localhost port=5433 user=anekazoo password=anekazoo123 dbname=anekazoo sslmode=disable")
    if err != nil {
        log.Fatal("Error connecting to the database:", err)
    }

	migrateDB()
	testDBConnection()

    err = db.Ping()
    if err != nil {
        log.Fatal("Database connection failed:", err)
    }

    log.Println("Database connected successfully")

    router := mux.NewRouter()
    router.HandleFunc("/animals", CreateAnimal).Methods("POST")
    router.HandleFunc("/animals", GetAllAnimals).Methods("GET")
    router.HandleFunc("/animals/{id}", GetAnimalByID).Methods("GET")
    router.HandleFunc("/animals/{id}", UpdateAnimal).Methods("PUT")
    router.HandleFunc("/animals/{id}", DeleteAnimal).Methods("DELETE")

    log.Println("Server is running on port 8080")
    http.ListenAndServe(":8080", router)
}

func CreateAnimal(w http.ResponseWriter, r *http.Request) {
    var animal Animal
    err := json.NewDecoder(r.Body).Decode(&animal)
    if err != nil {
        http.Error(w, "Invalid input", http.StatusBadRequest)
        return
    }

    query := `INSERT INTO animals (name, class, legs) VALUES ($1, $2, $3) RETURNING id`
    err = db.QueryRow(query, animal.Name, animal.Class, animal.Legs).Scan(&animal.ID)

    if err != nil {
        if pqErr, ok := err.(*pq.Error); ok {
            if pqErr.Code == "23505" {
                http.Error(w, "Animal with this name already exists", http.StatusConflict)
                return
            }
        }
        http.Error(w, "Could not create animal", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(animal)
}

func GetAllAnimals(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query("SELECT id, name, class, legs FROM animals")
    if err != nil {
        http.Error(w, "Could not fetch animals", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    animals := []Animal{}
    for rows.Next() {
        var animal Animal
        if err := rows.Scan(&animal.ID, &animal.Name, &animal.Class, &animal.Legs); err != nil {
            http.Error(w, "Could not scan animal", http.StatusInternalServerError)
            return
        }
        animals = append(animals, animal)
    }

    if len(animals) == 0 {
        http.Error(w, "No animals found", http.StatusNotFound)
        return
    }

    json.NewEncoder(w).Encode(animals)
}

func GetAnimalByID(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    var animal Animal
    query := `SELECT id, name, class, legs FROM animals WHERE id = $1`
    err := db.QueryRow(query, id).Scan(&animal.ID, &animal.Name, &animal.Class, &animal.Legs)

    if err != nil {
        http.Error(w, "Animal not found", http.StatusNotFound)
        return
    }

    json.NewEncoder(w).Encode(animal)
}

func UpdateAnimal(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    var animal Animal
    err := json.NewDecoder(r.Body).Decode(&animal)
    if err != nil {
        http.Error(w, "Invalid input", http.StatusBadRequest)
        return
    }

    query := `UPDATE animals SET name = $1, class = $2, legs = $3 WHERE id = $4`
    res, err := db.Exec(query, animal.Name, animal.Class, animal.Legs, id)
    if err != nil {
        http.Error(w, "Could not update animal", http.StatusInternalServerError)
        return
    }

    rowsAffected, _ := res.RowsAffected()
    if rowsAffected == 0 {
        http.Error(w, "Animal not found", http.StatusNotFound)
        return
    }

    w.WriteHeader(http.StatusOK)
}

func DeleteAnimal(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    query := `DELETE FROM animals WHERE id = $1`
    res, err := db.Exec(query, id)
    if err != nil {
        http.Error(w, "Could not delete animal", http.StatusInternalServerError)
        return
    }

    rowsAffected, _ := res.RowsAffected()
    if rowsAffected == 0 {
        http.Error(w, "Animal not found", http.StatusNotFound)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}