package main

import (
	"database/sql"
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pusher/pusher-http-go/v5"
)

var client = pusher.Client{
	AppID:   "1786859",
	Key:     "63a498aaf37284809f60",
	Secret:  "915d0a95f140273344e3",
	Cluster: "ap3",
	Secure:  true,
}

type Photo struct {
	ID  int64  `json:"id"`
	Src string `json:"src"`
}

type PhotoCollection struct {
	Photos []Photo `json:"items"`
}

func getPhotos(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		rows, err := db.Query("SELECT * FROM photos")

		if err != nil {
			panic(err)
		}
		defer rows.Close()

		result := PhotoCollection{}
		for rows.Next() {
			photo := Photo{}
			err2 := rows.Scan(&photo.ID, &photo.Src)
			if err2 != nil {
				panic(err2)
			}
			result.Photos = append(result.Photos, photo)
		}
		return c.JSON(http.StatusOK, result)
	}
}

func uploadPhoto(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		file, err := c.FormFile("file")

		if err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		filePath := "./public/uploads/" + file.Filename
		fileSrc := "http://127.0.0.1:9000/uploads/" + file.Filename
		dst, err := os.Create(filePath)
		if err != nil {
			panic(err)
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			panic(err)
		}

		stmt, err := db.Prepare("INSERT INTO photos (src) VALUES(?)")
		if err != nil {
			panic(err)
		}
		defer stmt.Close()
		result, err := stmt.Exec(fileSrc)
		if err != nil {
			panic(err)
		}
		insertedId, err := result.LastInsertId()
		if err != nil {
			panic(err)
		}
		photo := Photo{
			Src: fileSrc,
			ID:  insertedId,
		}

		client.Trigger("my-channel", "my-event", photo)

		return c.JSON(http.StatusOK, photo)

	}
}

func initialiseDatabase(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil || db == nil {
		panic("Error connection to database")
	}
	return db
}

func migrateDatabase(db *sql.DB) {
	sql := `        CREATE TABLE IF NOT EXISTS photos(                id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,                src VARCHAR NOT NULL        );    `
	_, err := db.Exec(sql)
	if err != nil {
		panic(err)
	}
}

func main() {
	db := initialiseDatabase("database/database.sqlite")
	migrateDatabase(db)
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.File("/", "public/index.html")
	e.GET("/photos", getPhotos(db))
	e.POST("/photos", uploadPhoto(db))
	e.Static("/uploads", "public/uploads")
	e.Logger.Fatal(e.Start(":9000"))

}
