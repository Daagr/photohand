package photohand

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dchesterton/goexiv"
	_ "github.com/mattn/go-sqlite3"
	uuid "github.com/satori/go.uuid"
	"gopkg.in/gographics/imagick.v3/imagick"
)

type Ph struct {
	Uuid              string
	Path, Filename    string
	Width, Height     int
	Artist, Copyright string
	F                 float32
	ISO               int
	Time              float32
	ExposureBias      float32
	FocalLength       float32
	Orientation       string
	Camera, Lens      string
	DateTime          string
	Rating            int
	Ext               string
}

var db *sql.DB
var resizes chan struct{}

const (
	Datadir ConfKey = iota + 1
	Picsdir
)

type ConfKey uint
type Conf struct {
	Key   ConfKey
	Value string
}

var datadir = "./data/"
var picsdir = ""

func Init(conf ...Conf) {
	for _, c := range conf {
		switch c.Key {
		case Datadir:
			if c.Value != "" {
				datadir = c.Value
			}
		case Picsdir:
			if c.Value != "" {
				picsdir = c.Value
			}
		default:
			log.Panicln("Unknown config item")
		}
	}
	// TODO: parallel resizes flag

	resizes = make(chan struct{}, 2)
	if picsdir == "" {
		log.Fatalln("TODO picsdir to ~/Pictures")
	}
	//log.Println("p:", picsdir, "d:", datadir)

	err := os.MkdirAll(filepath.Join(datadir, "thumbs"), os.ModePerm)
	err = os.MkdirAll(filepath.Join(datadir, "mids"), os.ModePerm)
	db_, err := sql.Open("sqlite3", filepath.Join(datadir, "db.db"))
	if err != nil {
		log.Fatal("Failed to open db", err)
	}
	db = db_
	InitDb()
	Update()
}

func InitDb() {
	db.Exec(`
	CREATE TABLE IF NOT EXISTS photos(uuid, path, filename, width, height, artist, copyright, f, iso, time, exposurebias, focallength, orientation, camera, lens, datetime, rating, ext);
	`)
}

func fraction(s string) float32 {
	num := 0
	den := 0

	n, err := fmt.Sscanf(s, "%d/%d", &num, &den)
	// log.Println(s, ":", num, "/", den, "??", n, err)
	if err != nil || n != 2 {
		return 0
	}
	return float32(num) / float32(den)
}

func scanPh(r *sql.Rows) (p Ph, err error) {
	err = r.Scan(&p.Uuid, &p.Path, &p.Filename, &p.Width, &p.Height, &p.Artist, &p.Copyright,
		&p.F, &p.ISO, &p.Time, &p.ExposureBias, &p.FocalLength, &p.Orientation,
		&p.Camera, &p.Lens, &p.DateTime, &p.Rating, &p.Ext)
	return p, err
}

func scanPhRow(r *sql.Row) (p Ph, err error) {
	err = r.Scan(&p.Uuid, &p.Path, &p.Filename, &p.Width, &p.Height, &p.Artist, &p.Copyright,
		&p.F, &p.ISO, &p.Time, &p.ExposureBias, &p.FocalLength, &p.Orientation,
		&p.Camera, &p.Lens, &p.DateTime, &p.Rating, &p.Ext)
	return p, err
}

func QueryPhs(query string, args ...interface{}) ([]Ph, error) {
	phs := make([]Ph, 0)
	rows, err := db.Query(query, args...)
	defer rows.Close()
	if err != nil {
		return phs, err
	}
	for rows.Next() {
		ph, err := scanPh(rows)
		if err != nil {
			return phs, err
		}
		phs = append(phs, ph)
	}
	err = rows.Err()
	return phs, err
}

func QueryPh(query string, args ...interface{}) (Ph, error) {
	row := db.QueryRow(query, args...)
	p, err := scanPhRow(row)
	return p, err
}

func AddInfo(ph Ph) {
	var unneeded sql.NullString
	err := db.QueryRow("SELECT filename FROM photos WHERE path == ?", ph.Path).Scan(&unneeded)
	if err != sql.ErrNoRows {
		return
	}

	ph.Ext = strings.TrimLeft(strings.ToLower(filepath.Ext(ph.Path)), ".")

	img, err := goexiv.Open(ph.Path)
	if err != nil {
		log.Println("Error while opening", ph.Path, err)
		return
	}
	err = img.ReadMetadata()
	if err != nil {
		log.Println("Error reading metadata", ph.Path, err)
		return
	}
	data := img.GetExifData()
	width, _ := data.FindKey("Exif.Image.ImageWidth")
	if width != nil {
		if v, err := strconv.Atoi(width.String()); err == nil {
			ph.Width = v
		}
	}
	width, _ = data.FindKey("Exif.Photo.PixelXDimension")
	if width != nil {
		if v, err := strconv.Atoi(width.String()); err == nil {
			if ph.Width == 0 || ph.Width == v {
				ph.Width = v
			} else if v == 0 && ph.Width != 0 {

			} else {
				log.Panicln("Image has multiple sizes")
			}
		}
	}
	length, _ := data.FindKey("Exif.Image.ImageLength")
	if length != nil {
		if v, err := strconv.Atoi(length.String()); err == nil {
			ph.Height = v
		}
	}
	length, _ = data.FindKey("Exif.Photo.PixelYDimension")
	if width != nil {
		if v, err := strconv.Atoi(length.String()); err == nil {
			if ph.Height == 0 || ph.Height == v {
				ph.Height = v
			} else if v == 0 && ph.Height != 0 {

			} else {
				log.Panicln("Image has multiple sizes")
			}
		}
	}

	datum, _ := data.FindKey("Exif.Image.Artist")
	if datum != nil {
		ph.Artist = strings.TrimSpace(datum.String())
	}

	datum, _ = data.FindKey("Exif.Image.Copyright")
	if datum != nil {
		ph.Copyright = strings.TrimSpace(datum.String())
	}

	datum, _ = data.FindKey("Exif.Image.Model")
	if datum != nil {
		ph.Camera = strings.TrimSpace(datum.String())
	}

	datum, _ = data.FindKey("Exif.Photo.LensModel")
	if datum != nil {
		ph.Lens = strings.TrimSpace(datum.String())
	}

	datum, _ = data.FindKey("Exif.Photo.FNumber")
	if datum != nil {
		ph.F = fraction(strings.TrimSpace(datum.String()))
	}

	datum, _ = data.FindKey("Exif.Photo.ISOSpeedRatings")
	if datum != nil {
		if v, err := strconv.Atoi(strings.TrimSpace(datum.String())); err == nil {
			ph.ISO = v
		}
	}

	datum, _ = data.FindKey("Exif.Photo.ExposureTime")
	if datum != nil {
		ph.Time = fraction(strings.TrimSpace(datum.String()))
	}

	datum, _ = data.FindKey("Exif.Photo.ExposureBiasValue")
	if datum != nil {
		ph.ExposureBias = fraction(strings.TrimSpace(datum.String()))
	}

	datum, _ = data.FindKey("Exif.Photo.FocalLength")
	if datum != nil {
		ph.FocalLength = fraction(strings.TrimSpace(datum.String()))
	}

	datum, _ = data.FindKey("Exif.Image.Orientation")
	if datum != nil {
		ph.Orientation = strings.TrimSpace(datum.String())
	}

	// TODO: doesn't seem correct
	datum, _ = data.FindKey("Exif.Image.DateTime")
	if datum != nil {
		ph.DateTime = strings.TrimSpace(datum.String())
	}

	// TODO: FocusDistance
	// TODO: some rotated images give opposite height/width

	ph.Uuid = uuid.Must(uuid.NewV4()).String()

	_, err = db.Exec(
		"INSERT INTO photos VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		ph.Uuid, ph.Path, ph.Filename, ph.Width, ph.Height, ph.Artist, ph.Copyright,
		ph.F, ph.ISO, ph.Time, ph.ExposureBias, ph.FocalLength, ph.Orientation,
		ph.Camera, ph.Lens, ph.DateTime, ph.Rating, ph.Ext)

	if err != nil {
		log.Fatalln("While adding a photo:", err)
	}
}

func Update() {
	filepath.Walk(picsdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("Error in walk: ", err)
			return err
		}
		if info.IsDir() {
			return nil
		}

		ph := Ph{}
		ph.Path = path
		ph.Filename = filepath.Base(path)

		AddInfo(ph)
		// log.Println("Found file: ", path)
		return nil
	})
}

func CreateThumb(uuid string, filename string) error {
	return CreateScaled(uuid, filename, 200)
}

func CreateMid(uuid string, filename string) error {
	return CreateScaled(uuid, filename, 1000)
}

func CreateScaled(uuid string, filename string, targetheight uint) error {
	resizes <- struct{}{}
	defer func() {
		<-resizes
	}()

	// Make sure the file doesn't exist (for example created while waiting)
	_, err := os.Stat(filename)
	if err != nil {
		return nil
	}
	if !os.IsExist(err) {
		log.Println("Unexpected error while checking file existence")
		return err
	}

	p, err := QueryPh("SELECT * FROM photos WHERE uuid = ?", uuid)
	if err != nil {
		log.Println("No image with id", uuid, "while trying to rescale it")
		return err
	}
	log.Println("Creating", targetheight, "high version of", p.Path)
	mw := imagick.NewMagickWand()
	err = mw.ReadImage(p.Path)
	if err != nil {
		return err
	}
	err = mw.AutoOrientImage()
	if err != nil {
		return err
	}
	height := float64(mw.GetImageHeight())
	width := float64(mw.GetImageWidth())
	nwidth := uint(width / height * float64(targetheight))
	if nwidth > 2*targetheight {
		nwidth = 2 * targetheight
	}
	// log.Println(height, width, nwidth)

	err = mw.ThumbnailImage(nwidth, targetheight)
	if err != nil {
		return err
	}
	err = mw.WriteImage(filename)
	return err
}
