FROM dock0/arch

RUN pacman -Syu --noconfirm --needed exiv2 imagemagick libmagick base-devel openexr go ghostscript libraw librsvg libwebp libwmf openjpeg2 pango

ENV GOPATH /go

RUN go get gopkg.in/gographics/imagick.v3/imagick && go get goji.io && go get github.com/satori/go.uuid && go get github.com/dchesterton/goexiv && go get github.com/mattn/go-sqlite3 && go get goji.io/pat

RUN mkdir -p /go/src/github.com/Daagr/photohand /data/thumbs /data/mids /pics

WORKDIR /go/src/github.com/Daagr/photohand

COPY . .

RUN go get -v .

RUN go install github.com/Daagr/photohand/cmd/photohand

EXPOSE 80

CMD /go/bin/photohand -host 0.0.0.0:80 -data /data -pics /pics
