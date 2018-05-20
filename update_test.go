package photohand

import (
	"log"
	"os"
	"testing"
)

func TestWa(t *testing.T) {
	os.Remove("./data/db.db")
	Init(Conf{Picsdir, "./imgs/"}, Conf{Datadir, "./data/"})
	InitDb()
	Update()
	phs, _ := QueryPhs("SELECT * FROM photos")
	log.Println(phs)
}
