package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var tmpl = template.Must(template.ParseFiles("templates/index.html"))
var collection *mongo.Collection

type URL struct {
	Code string `bson:"code"`
	Long string `bson:"long"`
}

func connectDB() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system env variables")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Mongo connection error:", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Mongo ping error:", err)
	}

	fmt.Println("Connected to MongoDB")
	collection = client.Database("urlshortener").Collection("urls")

	indexModel := mongo.IndexModel{
		Keys:    bson.M{"code": 1},
		Options: options.Index().SetUnique(true),
	}

	_, err = collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		log.Fatal("Failed to create unique index:", err)
	}
}

func generateCode(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])[:8]
}

func startsWithHTTP(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.Execute(w, nil)
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	longURL := r.FormValue("url")
	if longURL == "" {
		http.Error(w, "URL cannot be empty", http.StatusBadRequest)
		return
	}

	if !startsWithHTTP(longURL) {
		longURL = "https://" + longURL
	}

	_, err := url.ParseRequestURI(longURL)
	if err != nil {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	code := generateCode(longURL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existing URL
	err = collection.FindOne(ctx, bson.M{"code": code}).Decode(&existing)

	if err == mongo.ErrNoDocuments {
		_, insertErr := collection.InsertOne(ctx, URL{
			Code: code,
			Long: longURL,
		})
		if insertErr != nil {
			http.Error(w, "Database insert error", 500)
			return
		}
	} else if err != nil {
		http.Error(w, "Database find error", 500)
		return
	}

	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
	}

	shortURL := fmt.Sprintf("%s://%s/%s", proto, host, code)

	data := struct {
		ShortURL string
	}{
		ShortURL: shortURL,
	}

	tmpl.Execute(w, data)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result URL
	err := collection.FindOne(ctx, bson.M{"code": code}).Decode(&result)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, result.Long, http.StatusMovedPermanently)
}

func main() {
	connectDB()

	http.HandleFunc("/shorten", shortenHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			homeHandler(w, r)
		} else {
			redirectHandler(w, r)
		}
	})

	fmt.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}