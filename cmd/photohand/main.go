package main

import (
	"flag"

	"github.com/Daagr/photohand"
)

var host = flag.String("host", "127.0.0.1:8000", "Hostname:port to bind to")
var root = flag.String("root", "/", "Root on the server (useful with reverse proxys)")
var data = flag.String("data", "./data/", "Folder for database and thumbnails")
var pics = flag.String("pics", "", "Folder where pictures are found")

func main() {
	flag.Parse()
	photohand.Init(photohand.Conf{photohand.Picsdir, *pics}, photohand.Conf{photohand.Datadir, *data})
	photohand.Serve(*host, *root)
}
